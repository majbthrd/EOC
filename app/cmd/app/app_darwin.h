#import <Cocoa/Cocoa.h>

void run(bool firstTimeRun, bool startHidden);
void StartUI(const char *path);
void ShowUI();
void StopUI();
void quit();
void uiRequest(char *path);
void setWindowDelegate(void *window);
void showWindow(uintptr_t wndPtr);
void hideWindow(uintptr_t wndPtr);
void styleWindow(uintptr_t wndPtr);
void drag(uintptr_t wndPtr);
void doubleClick(uintptr_t wndPtr);
void handleConnectURL();
