//go:build darwin

package main

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Webkit -framework Cocoa
// #include "app_darwin.h"
// typedef const char cchar_t;
import "C"

import (
	"log/slog"
	"os"
	"unsafe"

	"github.com/majbthrd/EOC/app/version"
)

//export StartUI
func StartUI(path *C.cchar_t) {
	p := C.GoString(path)
	wv.Run(p)
	styleWindow(wv.webview.Window())
	C.setWindowDelegate(wv.webview.Window())
}

//export ShowUI
func ShowUI() {
	if wv.IsRunning() && wv.webview != nil {
		showWindow(wv.webview.Window())
	} else {
		root := C.CString("/")
		defer C.free(unsafe.Pointer(root))
		StartUI(root)
	}
}

//export StopUI
func StopUI() {
	wv.Terminate()
}

//export handleConnectURL
func handleConnectURL() {
//	handleConnectURLScheme()
}

// maybeMoveAndRestart: standalone executable, nothing to relocate.
func maybeMoveAndRestart() appMove {
	return AlreadyMoved
}

func handleExistingInstance(_ bool) {}

func osRun(_ func(), hasCompletedFirstRun, startHidden bool) {
	// Run the native macOS app
	// Note: this will block until the app is closed
	slog.Debug("starting native darwin event loop")
	C.run(C._Bool(hasCompletedFirstRun), C._Bool(startHidden))
}

func quit() {
	C.quit()
	os.Exit(0)
}

func logStartup() {
	exe, _ := os.Executable()
	slog.Info("starting EOC", "app", exe, "version", version.Version, "os", "darwin")
}

func hideWindow(ptr unsafe.Pointer) {
	C.hideWindow(C.uintptr_t(uintptr(ptr)))
}

func showWindow(ptr unsafe.Pointer) {
	C.showWindow(C.uintptr_t(uintptr(ptr)))
}

func styleWindow(ptr unsafe.Pointer) {
	C.styleWindow(C.uintptr_t(uintptr(ptr)))
}

func drag(ptr unsafe.Pointer) {
	C.drag(C.uintptr_t(uintptr(ptr)))
}

func doubleClick(ptr unsafe.Pointer) {
	C.doubleClick(C.uintptr_t(uintptr(ptr)))
}

func checkAndHandleExistingInstance(_ string) bool {
	return false
}

