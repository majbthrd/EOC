#import "app_darwin.h"
#import "menu.h"
#import <AppKit/AppKit.h>
#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#import <objc/runtime.h>

static WKWebView *FindWKWebView(NSView *root);

@interface AppDelegate : NSObject <NSApplicationDelegate, NSWindowDelegate, WKNavigationDelegate, WKUIDelegate>
@end

@implementation AppDelegate

- (void)applicationDidFinishLaunching:(NSNotification *)aNotification {
    // Application menu
    NSString *appName = @"EOC";

    NSMenu *mainMenu = [[NSMenu alloc] init];

    // App menu
    NSMenuItem *appMenuItem = [[NSMenuItem alloc] initWithTitle:appName
                                                        action:nil
                                                 keyEquivalent:@""];
    NSMenu *appMenu = [[NSMenu alloc] initWithTitle:appName];
    [appMenuItem setSubmenu:appMenu];
    [mainMenu addItem:appMenuItem];

    [appMenu addItemWithTitle:[NSString stringWithFormat:@"Quit %@", appName]
                       action:@selector(quit)
                keyEquivalent:@"q"];

    // Edit menu
    NSMenuItem *editMenuItem = [[NSMenuItem alloc] init];
    NSMenu *editMenu = [[NSMenu alloc] initWithTitle:@"Edit"];

    [editMenu addItemWithTitle:@"Undo"
                        action:@selector(undo:)
                 keyEquivalent:@"z"];
    [editMenu addItemWithTitle:@"Redo"
                        action:@selector(redo:)
                 keyEquivalent:@"Z"];
    [editMenu addItem:[NSMenuItem separatorItem]];
    [editMenu addItemWithTitle:@"Cut"
                        action:@selector(cut:)
                 keyEquivalent:@"x"];
    [editMenu addItemWithTitle:@"Copy"
                        action:@selector(copy:)
                 keyEquivalent:@"c"];
    [editMenu addItemWithTitle:@"Paste"
                        action:@selector(paste:)
                 keyEquivalent:@"v"];
    [editMenu addItemWithTitle:@"Select All"
                        action:@selector(selectAll:)
                 keyEquivalent:@"a"];

    [editMenuItem setSubmenu:editMenu];
    [mainMenu addItem:editMenuItem];

    // Window menu
    NSMenuItem *windowMenuItem = [[NSMenuItem alloc] init];
    NSMenu *windowMenu = [[NSMenu alloc] initWithTitle:@"Window"];
    [windowMenu addItemWithTitle:@"Minimize"
                          action:@selector(performMiniaturize:)
                   keyEquivalent:@"m"];
    [windowMenu addItemWithTitle:@"Zoom"
                          action:@selector(performZoom:)
                   keyEquivalent:@""];
    [windowMenu addItem:[NSMenuItem separatorItem]];
    [windowMenu addItemWithTitle:@"Bring All to Front"
                          action:@selector(arrangeInFront:)
                   keyEquivalent:@""];
    [windowMenuItem setSubmenu:windowMenu];
    [mainMenu addItem:windowMenuItem];
    [NSApp setWindowsMenu:windowMenu];

    [NSApp setMainMenu:mainMenu];

    // Launch the webview
    dispatch_async(dispatch_get_main_queue(), ^{
        StartUI("/");
    });

for (NSWindow *w in [NSApp windows]) {
    NSLog(@"window: %@ visible=%d", w, w.isVisible);
    [w makeKeyAndOrderFront:nil];
    w.level = NSFloatingWindowLevel;
}
}

- (BOOL)applicationShouldHandleReopen:(NSApplication *)sender hasVisibleWindows:(BOOL)hasVisibleWindows {
    [self openUI];
    return YES;
}

- (BOOL)windowShouldClose:(id)sender {
    [NSApp hide:nil];
    return NO;
}

- (NSApplicationTerminateReply)applicationShouldTerminate:(NSApplication *)sender {
    return NSTerminateNow;
}

- (void)openUI {
    ShowUI();
}

static void appLogInfo(NSString *msg) {
    NSLog(@"[eoc-app] %@", msg);
}

- (void)uiRequest:(NSString *)path {
    if (path == nil) {
        appLogInfo(@"app UI request for URL is missing");
        return;
    }
    appLogInfo([NSString stringWithFormat:@"app UI request for URL: %@", path]);
    StartUI([path UTF8String]);
}

- (void)quit {
    [NSApp terminate:nil];
}

#pragma mark - WKNavigationDelegate

- (void)webView:(WKWebView *)webView
decidePolicyForNavigationAction:(WKNavigationAction *)action
        decisionHandler:(void (^)(WKNavigationActionPolicy))handler
{
    NSURL *url = action.request.URL;
    if (action.navigationType == WKNavigationTypeLinkActivated) {
        NSString *host = [url.host lowercaseString];
        if ([host isEqualToString:@"localhost"] ||
            [host isEqualToString:@"127.0.0.1"]) {
            handler(WKNavigationActionPolicyCancel);
            NSString *path = url.path;
            if (path.length == 0) {
                path = @"/";
            }
            [self uiRequest:path];
            return;
        }

        [[NSWorkspace sharedWorkspace] openURL:url];
        handler(WKNavigationActionPolicyCancel);
        return;
    }
    handler(WKNavigationActionPolicyAllow);
}

- (nullable WKWebView *)webView:(WKWebView *)webView
   createWebViewWithConfiguration:(WKWebViewConfiguration *)configuration
             forNavigationAction:(WKNavigationAction *)action
                  windowFeatures:(WKWindowFeatures *)features
{
    NSURL *url = action.request.URL;
    if (url) {
        NSString *host = [url.host lowercaseString];
        if ([host isEqualToString:@"localhost"] ||
            [host isEqualToString:@"127.0.0.1"]) {
            return nil;
        }
        [[NSWorkspace sharedWorkspace] openURL:url];
    }
    return nil;
}

#pragma mark - WKUIDelegate

- (void)webView:(WKWebView *)webView
    runJavaScriptConfirmPanelWithMessage:(NSString *)message
    initiatedByFrame:(WKFrameInfo *)frame
    completionHandler:(void (^)(BOOL))completionHandler {

    NSAlert *alert = [[NSAlert alloc] init];
    [alert setMessageText:message];
    [alert addButtonWithTitle:@"Confirm"];
    [alert addButtonWithTitle:@"Cancel"];

    completionHandler([alert runModal] == NSAlertFirstButtonReturn);
}

#pragma mark - Context menu swizzle

+ (void)load {
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        [self swizzleWKWebViewContextMenu];
    });
}

+ (void)swizzleWKWebViewContextMenu {
    Class class = [WKWebView class];

    SEL originalSelector = @selector(willOpenMenu:withEvent:);
    SEL swizzledSelector = @selector(ollama_willOpenMenu:withEvent:);

    Method originalMethod = class_getInstanceMethod(class, originalSelector);
    Method swizzledMethod = class_getInstanceMethod(class, swizzledSelector);
    BOOL didAddMethod = class_addMethod(class, originalSelector,
                                       method_getImplementation(swizzledMethod),
                                       method_getTypeEncoding(swizzledMethod));

    if (didAddMethod) {
        class_replaceMethod(class,
                           swizzledSelector,
                           method_getImplementation(originalMethod),
                           method_getTypeEncoding(originalMethod));
    } else {
        method_exchangeImplementations(originalMethod, swizzledMethod);
    }
}

@end

@implementation WKWebView (OllamaContextMenu)
- (void)ollama_willOpenMenu:(NSMenu *)menu withEvent:(NSEvent *)event {
    [self ollama_willOpenMenu:menu withEvent:event];
    NSMutableArray *itemsToRemove = [NSMutableArray array];
    for (NSMenuItem *item in menu.itemArray) {
        if ([item.title containsString:@"Copy Link with Highlight"] ||
            [item.title containsString:@"Open Link in New Window"] ||
            [item.title containsString:@"Services"] ||
            [item.title containsString:@"Download Linked File"] ||
            [item.title containsString:@"Back"] ||
            [item.title containsString:@"Reload"] ||
            [item.title containsString:@"Refresh"] ||
            [item.title containsString:@"Open Link"] ||
            [item.title containsString:@"Copy Link"] ||
            [item.title containsString:@"Share"]) {
            [itemsToRemove addObject:item];
        }
    }
    for (NSMenuItem *item in itemsToRemove) {
        [menu removeItem:item];
    }

    // Inject custom menu items from the Go/webview side
    int customItemCount = menu_get_item_count();
    if (customItemCount > 0) {
        menuItem *customItems = (menuItem *)menu_get_items();
        if (customItems) {
            NSInteger insertIndex = 0;
            for (int i = 0; i < customItemCount; i++) {
                if (customItems[i].separator) {
                    [menu insertItem:[NSMenuItem separatorItem] atIndex:insertIndex++];
                } else if (customItems[i].label) {
                    NSString *label = [NSString stringWithUTF8String:customItems[i].label];
                    NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:label
                                                                  action:@selector(handleCustomMenuItem:)
                                                           keyEquivalent:@""];
                    [item setTarget:self];
                    [item setRepresentedObject:label];
                    [item setEnabled:customItems[i].enabled];
                    [menu insertItem:item atIndex:insertIndex++];
                }
            }
            if (insertIndex > 0 && menu.itemArray.count > insertIndex) {
                [menu insertItem:[NSMenuItem separatorItem] atIndex:insertIndex];
            }
        }
    }
}

- (void)handleCustomMenuItem:(NSMenuItem *)sender {
    NSString *label = [sender representedObject];
    if (label) {
        menu_handle_selection((char *)[label UTF8String]);
    }
}
@end

#pragma mark - C entry points

AppDelegate *appDelegate;

void run(bool firstTimeRun, bool startHidden) {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    appDelegate = [[AppDelegate alloc] init];
    [NSApp setDelegate:appDelegate];
    [NSApp run];
    StopUI();
}

void quit() {
    dispatch_async(dispatch_get_main_queue(), ^{
        [appDelegate quit];
    });
}

void uiRequest(char *path) {
    NSString *p = [NSString stringWithUTF8String:path];
    dispatch_async(dispatch_get_main_queue(), ^{
        [appDelegate uiRequest:p];
    });
}

void setWindowDelegate(void *window) {
    NSWindow *w = (__bridge NSWindow *)window;
    [w setDelegate:appDelegate];
    WKWebView *webView = FindWKWebView(w.contentView);
    if (webView) {
        webView.navigationDelegate = appDelegate;
        webView.UIDelegate = appDelegate;
    }
}

void hideWindow(uintptr_t wndPtr) {
    NSWindow *w = (__bridge NSWindow *)wndPtr;
    [w orderOut:nil];
}

void showWindow(uintptr_t wndPtr) {
    NSWindow *w = (__bridge NSWindow *)wndPtr;
    if (!w) return; // Guard against a nil pointer

    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp unhide:nil];
        
        // Fallback check: If the cross-platform library initialized 
        // the frame with a size of 0x0, force standard dimensions.
        if (w.frame.size.width <= 0 || w.frame.size.height <= 0) {
            [w setFrame:NSMakeRect(0, 0, 1024, 768) display:YES];
        }
        
        // Center and map the window to the top-level viewport
        [w center];
        [w makeKeyAndOrderFront:nil];
        
        // Force foreground application focus
        if (@available(macOS 14.0, *)) {
            [NSApp activate];
        } else {
            [NSApp activateIgnoringOtherApps:YES];
        }
    });
}


void styleWindow(uintptr_t wndPtr) {
    NSWindow *w = (__bridge NSWindow *)wndPtr;
    if (!w) return;
    w.titleVisibility = NSWindowTitleVisible;
    w.titlebarAppearsTransparent = NO;
}

void drag(uintptr_t wndPtr) {
    NSWindow *w = (__bridge NSWindow *)wndPtr;
    if (!w) return;
    NSPoint mouseLoc = [NSEvent mouseLocation];
    NSPoint locInWindow = [w convertPointFromScreen:mouseLoc];

    NSEvent *e = [NSEvent mouseEventWithType:NSEventTypeLeftMouseDown
                                    location:locInWindow
                                modifierFlags:0
                                    timestamp:NSTimeIntervalSince1970
                                windowNumber:[w windowNumber]
                                     context:nil
                                 eventNumber:0
                                  clickCount:1
                                    pressure:1.0];
    [w performWindowDragWithEvent:e];
}

void doubleClick(uintptr_t wndPtr) {
    NSWindow *w = (__bridge NSWindow *)wndPtr;
    if (!w) return;

    NSString *action =
        [[NSUserDefaults standardUserDefaults] stringForKey:@"AppleActionOnDoubleClick"];
    if ([action isEqualToString:@"Minimize"]) {
        [w performMiniaturize:nil];
    } else {
        [w performZoom:nil];
    }
}

#pragma mark - Helpers

static WKWebView *FindWKWebView(NSView *root) {
    if ([root isKindOfClass:[WKWebView class]]) {
        return (WKWebView *)root;
    }
    for (NSView *child in root.subviews) {
        WKWebView *found = FindWKWebView(child);
        if (found) {
            return found;
        }
    }
    return nil;
}
