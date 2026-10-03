//go:build windows || darwin

package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/majbthrd/EOC/app/version"
	"golang.org/x/sys/windows"
)

var (
	msgPumpU32      = windows.NewLazySystemDLL("User32.dll")
	msgPumpK32      = windows.NewLazySystemDLL("Kernel32.dll")
	pGetMessage     = msgPumpU32.NewProc("GetMessageW")
	pTranslateMsg   = msgPumpU32.NewProc("TranslateMessage")
	pDispatchMsg    = msgPumpU32.NewProc("DispatchMessageW")
	pDefWindowProc  = msgPumpU32.NewProc("DefWindowProcW")
	pCreateWinEx    = msgPumpU32.NewProc("CreateWindowExW")
	pRegisterClass  = msgPumpU32.NewProc("RegisterClassExW")
	pUnregisterCls  = msgPumpU32.NewProc("UnregisterClassW")
	pShowWindow     = msgPumpU32.NewProc("ShowWindow")
	pUpdateWindow   = msgPumpU32.NewProc("UpdateWindow")
	pPostQuit       = msgPumpU32.NewProc("PostQuitMessage")
	pGetModuleHnd   = msgPumpK32.NewProc("GetModuleHandleW")
	pLoadCursor     = msgPumpU32.NewProc("LoadCursorW")

	u32                  = windows.NewLazySystemDLL("User32.dll")
	pBringWindowToTop    = u32.NewProc("BringWindowToTop")
	pSendMessage         = u32.NewProc("SendMessageA")
	pGetSystemMetrics    = u32.NewProc("GetSystemMetrics")
	pGetWindowRect       = u32.NewProc("GetWindowRect")
	pSetWindowPos        = u32.NewProc("SetWindowPos")
	pSetForegroundWindow = u32.NewProc("SetForegroundWindow")
	pSetActiveWindow     = u32.NewProc("SetActiveWindow")
	pIsIconic            = u32.NewProc("IsIconic")

	appPath         = filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Ollama")
	appLogPath      = filepath.Join(os.Getenv("LOCALAPPDATA"), "Ollama", "app.log")
	startupShortcut = filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "Ollama.lnk")
	ollamaPath      string
	DesktopAppName  = "ollama app.exe"
)

const (
	CS_HREDRAW = 0x0002
	CS_VREDRAW = 0x0001
	IDC_ARROW  = 32512
)

// Minimal WNDCLASSEX
type wndClassEx struct {
	Size     uint32
	Style    uint32
	WndProc  uintptr
	ClsExtra int32
	WndExtra int32
	Instance windows.Handle
	Icon     windows.Handle
	Cursor   windows.Handle
	Background windows.Handle
	MenuName *uint16
	ClassName *uint16
	IconSm   windows.Handle
}

var (
	ownerWindow windows.Handle // hidden window the webview parents to
	msgClass    wndClassEx
)

// msgLoopWndProc – all we need is the default handler.
func msgLoopWndProc(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	// If we ever need to intercept WM_COPYDATA for single-instance
	// or URL-scheme routing, do it here.  For now, pass everything through.
	ret, _, _ := pDefWindowProc.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

// initMsgPump registers a window class, creates a hidden owner window,
// and locks the calling goroutine to an OS thread (required for Win32).
func initMsgPump() error {
	runtime.LockOSThread()

	clsName, _ := windows.UTF16PtrFromString("OllamaMsgPump")
	winName, _ := windows.UTF16PtrFromString("")

	instance, _, _ := pGetModuleHnd.Call(0)
	if instance == 0 {
		return fmt.Errorf("GetModuleHandleW failed")
	}

	cursor, _, _ := pLoadCursor.Call(0, uintptr(IDC_ARROW))
	if cursor == 0 {
		return fmt.Errorf("LoadCursorW failed")
	}

	msgClass = wndClassEx{
		Style:      CS_HREDRAW | CS_VREDRAW,
		WndProc:    windows.NewCallback(msgLoopWndProc),
		Instance:   windows.Handle(instance),
		Cursor:     windows.Handle(cursor),
		Background: windows.Handle(6), // COLOR_WINDOW + 1
		ClassName:  clsName,
	}
	msgClass.Size = uint32(unsafe.Sizeof(msgClass))

	// RegisterClassExW returns an ATOM (non-zero on success, 0 on failure)
	atom, _, _ := pRegisterClass.Call(uintptr(unsafe.Pointer(&msgClass)))
	if atom == 0 {
		return fmt.Errorf("RegisterClassExW failed")
	}

	// CreateWindowExW returns an HWND (non-zero on success, 0 on failure)
	hwnd, _, _ := pCreateWinEx.Call(
		0,
		uintptr(unsafe.Pointer(clsName)),
		uintptr(unsafe.Pointer(winName)),
		0, 0, 0, 0, 0, 0, 0,
		uintptr(instance),
		0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowExW failed")
	}
	ownerWindow = windows.Handle(hwnd)

	pShowWindow.Call(hwnd, uintptr(SW_HIDE))
	pUpdateWindow.Call(hwnd)
	return nil
}

// runMsgPump blocks until PostQuitMessage is called.
func runMsgPump() {
	m := struct {
		Hwnd   windows.Handle
		Msg    uint32
		WParam uintptr
		LParam uintptr
		Time   uint32
		Pt     [2]int32
		Priv   uint32
	}{}
	for {
		ret, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) == -1 {
			slog.Error("GetMessage failed")
			return
		}
		if int32(ret) == 0 { // WM_QUIT
			return
		}
		pTranslateMsg.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMsg.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// quitMsgPump posts WM_QUIT to unblock runMsgPump.
func quitMsgPump() {
	pPostQuit.Call(0)
}

// cleanupMsgPump unregisters the window class (call before os.Exit).
func cleanupMsgPump() {
	if ownerWindow != 0 {
		clsName, _ := windows.UTF16PtrFromString("OllamaMsgPump")
		instance, _, _ := pGetModuleHnd.Call(0)
		pUnregisterCls.Call(uintptr(unsafe.Pointer(clsName)), uintptr(instance))
	}
}

func init() {
	// With alternate install location use executable location
	exe, err := os.Executable()
	if err != nil {
		slog.Warn("error discovering executable directory", "error", err)
	} else {
		appPath = filepath.Dir(exe)
	}
	ollamaPath = filepath.Join(appPath, "ollama.exe")

	// Handle developer mode (go run ./cmd/app)
	if _, err := os.Stat(ollamaPath); err != nil {
		pwd, err := os.Getwd()
		if err != nil {
			slog.Warn("missing ollama.exe and failed to get pwd", "error", err)
			return
		}
		distAppPath := filepath.Join(pwd, "dist", "windows-"+runtime.GOARCH)
		distOllamaPath := filepath.Join(distAppPath, "ollama.exe")
		if _, err := os.Stat(distOllamaPath); err == nil {
			slog.Info("detected developer mode")
			appPath = distAppPath
			ollamaPath = distOllamaPath
		}
	}
}

func maybeMoveAndRestart() appMove {
	return 0
}

// handleExistingInstance checks for existing instances and optionally focuses them
func handleExistingInstance(startHidden bool) {}

func installSymlink() {}

type appCallbacks struct {
	shutdown func()
}

var app = &appCallbacks{}

func (ac *appCallbacks) UIRun(path string)  { wv.Run(path) }
func (*appCallbacks) UIShow()              { if wv.webview != nil { showWindow(wv.webview.Window()) } else { wv.Run("/") } }
func (*appCallbacks) UITerminate()         { wv.Terminate() }
func (*appCallbacks) UIRunning() bool      { return wv.IsRunning() }

func (app *appCallbacks) Quit() {
	wv.Terminate()
	quitMsgPump()
}

// TODO - reconcile with above for consistency between mac/windows
func quit() {
	wv.Terminate()
	quitMsgPump()
	os.Exit(0)
}

func (app *appCallbacks) DoUpdate() {
	// Safeguard in case we have requests in flight that need to drain...
	slog.Info("Waiting for server to shutdown")

	app.shutdown()
}

func osRun(shutdown func(), hasCompletedFirstRun, startHidden bool) {
	app.shutdown = shutdown

	if err := initMsgPump(); err != nil {
		log.Fatalf("Failed to start message pump: %s", err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signals
		slog.Debug("shutting down due to signal")
		quitMsgPump()
		wv.Terminate()
	}()

	ptr := wv.Run("/")

	// Set the window icon (load directly from assets, skip the tray)
	if ptr != nil {
		setWebViewIcon(ptr)
	}

	centerWindow(ptr)

	runMsgPump()
}

// setWebViewIcon loads the .ico from embedded assets and applies it to the
// webview HWND.
func setWebViewIcon(ptr unsafe.Pointer) {
/*
	iconPath, err := iconBytesToFilePath(assets.GetIconBytes("tray.ico")) // adjust to your actual helper
	if err != nil {
		slog.Warn("could not load icon for window", "err", err)
		return
	}
	srcPtr, _ := windows.UTF16PtrFromString(iconPath)
	const (
		IMAGE_ICON      = 1
		LR_LOADFROMFILE = 0x10
		LR_DEFAULTSIZE  = 0x40
		ICON_SMALL      = 0
		ICON_BIG        = 1
		WM_SETICON      = 0x0080
	)
	pLoadImage := u32.NewProc("LoadImageW")
	h, _, _ := pLoadImage.Call(0, uintptr(unsafe.Pointer(srcPtr)), uintptr(IMAGE_ICON), 0, 0,
		uintptr(LR_LOADFROMFILE|LR_DEFAULTSIZE))
	if h != 0 {
		hwnd := uintptr(ptr)
		pSendMessage.Call(hwnd, uintptr(WM_SETICON), uintptr(ICON_SMALL), h)
		pSendMessage.Call(hwnd, uintptr(WM_SETICON), uintptr(ICON_BIG), h)
	}
*/
}

func LaunchNewApp() {
}

func logStartup() {
	slog.Info("starting Ollama", "app", appPath, "version", version.Version)
}

const (
	SW_HIDE        = 0  // Hides the window
	SW_SHOW        = 5  // Shows window in its current size/position
	SW_SHOWNA      = 8  // Shows without activating
	SW_MINIMIZE    = 6  // Minimizes the window
	SW_RESTORE     = 9  // Restores to previous size/position
	SW_SHOWDEFAULT = 10 // Sets show state based on program state
	SM_CXSCREEN    = 0
	SM_CYSCREEN    = 1
	HWND_TOP       = 0
	SWP_NOSIZE     = 0x0001
	SWP_NOMOVE     = 0x0002
	SWP_NOZORDER   = 0x0004
	SWP_SHOWWINDOW = 0x0040

	// Menu constants
	MF_STRING     = 0x00000000
	MF_SEPARATOR  = 0x00000800
	MF_GRAYED     = 0x00000001
	TPM_RETURNCMD = 0x0100
)

// POINT structure for cursor position
type POINT struct {
	X int32
	Y int32
}

// Rect structure for GetWindowRect
type Rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func centerWindow(ptr unsafe.Pointer) {
	hwnd := uintptr(ptr)
	if hwnd == 0 {
		return
	}

	var rect Rect
	pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))

	screenWidth, _, _ := pGetSystemMetrics.Call(uintptr(SM_CXSCREEN))
	screenHeight, _, _ := pGetSystemMetrics.Call(uintptr(SM_CYSCREEN))

	windowWidth := rect.Right - rect.Left
	windowHeight := rect.Bottom - rect.Top

	x := (int32(screenWidth) - windowWidth) / 2
	y := (int32(screenHeight) - windowHeight) / 2

	// Ensure the window is not positioned off-screen
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	// First call: nudge width by 1px to guarantee a WM_SIZE is delivered.
	// WebView2 on some Windows builds does not start rendering until it
	// receives WM_SIZE, and SetWindowPos with identical dimensions is
	// silently optimised away by the window manager.
	pSetWindowPos.Call(
		hwnd,
		uintptr(HWND_TOP),
		uintptr(x),
		uintptr(y),
		uintptr(windowWidth+1), // temporary +1 to force WM_SIZE
		uintptr(windowHeight),
		uintptr(SWP_SHOWWINDOW),
	)

	// Second call: restore the intended dimensions (triggers a second
	// WM_SIZE so WebView2 lays out to the correct size).
	pSetWindowPos.Call(
		hwnd,
		uintptr(HWND_TOP),
		uintptr(x),
		uintptr(y),
		uintptr(windowWidth),
		uintptr(windowHeight),
		uintptr(SWP_SHOWWINDOW),
	)
}

func showWindow(ptr unsafe.Pointer) {
	hwnd := uintptr(ptr)
	if hwnd == 0 {
		return
	}

	// Re-apply icon if needed (optional; skip if you already set it at startup)
	// … same LoadImage + WM_SETICON as setWebViewIcon …

	isMinimized, _, _ := pIsIconic.Call(hwnd)
	if isMinimized != 0 {
		pShowWindow.Call(hwnd, uintptr(SW_RESTORE))
	}
	pShowWindow.Call(hwnd, uintptr(SW_SHOW))
	pBringWindowToTop.Call(hwnd)
	pSetForegroundWindow.Call(hwnd)
	pSetActiveWindow.Call(hwnd)
	pSetWindowPos.Call(hwnd, uintptr(HWND_TOP), 0, 0, 0, 0,
		uintptr(SWP_NOSIZE|SWP_NOMOVE|SWP_SHOWWINDOW))
}

// HideWindow hides the application window
func hideWindow(ptr unsafe.Pointer) {
	hwnd := uintptr(ptr)
	if hwnd != 0 {
		pShowWindow.Call(
			hwnd,
			uintptr(SW_HIDE),
		)
	}
}

func drag(ptr unsafe.Pointer) {}

func doubleClick(ptr unsafe.Pointer) {}

// checkAndHandleExistingInstance checks if another instance is running and sends the URL to it
func checkAndHandleExistingInstance(urlSchemeRequest string) bool {
	// No existing instance, we'll handle it ourselves
	return false
}
