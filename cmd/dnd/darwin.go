//go:build darwin
// +build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework AppKit
#include <Foundation/Foundation.h>
#include <AppKit/AppKit.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// ===== 1. Функция отправки сообщения =====
static void sendToSocket(const char* msg, const char* sockPath) {
    int clientSock = socket(AF_UNIX, SOCK_STREAM, 0);
    if (clientSock == -1) return;

    struct sockaddr_un addr;
    memset(&addr, 0, sizeof(addr));
    addr.sun_family = AF_UNIX;
    strncpy(addr.sun_path, sockPath, sizeof(addr.sun_path)-1);

    if (connect(clientSock, (struct sockaddr*)&addr, sizeof(addr)) == -1) {
        close(clientSock);
        return;
    }

    write(clientSock, msg, strlen(msg));
    close(clientSock);
}

// ===== 2. Делегат окна =====
@interface WindowDelegate : NSObject <NSWindowDelegate> {
    const char* sockPath;
}
- (id)initWithSocketPath:(const char*)path;
@end

@implementation WindowDelegate
- (id)initWithSocketPath:(const char*)path {
    self = [super init];
    if (self) {
        sockPath = strdup(path);
    }
    return self;
}

- (void)windowWillClose:(NSNotification *)notification {
    sendToSocket("", sockPath);
}

- (void)dealloc {
    free((void*)sockPath);
    [super dealloc];
}
@end

// ===== 3. View для перетаскивания =====
@interface DragView : NSView <NSDraggingDestination> {
    const char* sockPath;
}
- (id)initWithFrame:(NSRect)frameRect socketPath:(const char*)path;
@end

@implementation DragView
- (id)initWithFrame:(NSRect)frameRect socketPath:(const char*)path {
    self = [super initWithFrame:frameRect];
    if (self) {
        sockPath = strdup(path);
        [self registerForDraggedTypes:@[NSPasteboardTypeFileURL]];
    }
    return self;
}

- (NSDragOperation)draggingEntered:(id<NSDraggingInfo>)sender {
    return NSDragOperationCopy;
}

- (BOOL)performDragOperation:(id<NSDraggingInfo>)sender {
    NSPasteboard *pboard = [sender draggingPasteboard];
    NSArray *urls = [pboard readObjectsForClasses:@[[NSURL class]]
                                        options:@{NSPasteboardURLReadingFileURLsOnlyKey: @YES}];

    NSMutableString *combinedPaths = [NSMutableString string];
    for (NSURL *url in urls) {
        if ([combinedPaths length] > 0) [combinedPaths appendString:@"\n"];
        [combinedPaths appendString:[url path]];
    }

    if ([combinedPaths length] > 0) {
        sendToSocket([combinedPaths UTF8String], sockPath);
    }
    return YES;
}

- (void)dealloc {
    free((void*)sockPath);
    [super dealloc];
}
@end

// ===== 4. Главный делегат =====
@interface AppDelegate : NSObject <NSApplicationDelegate> {
    NSString* windowTitle;
    const char* sockPath;
    WindowDelegate* windowDelegate;
}
- (void)setWindowTitle:(NSString*)title;
- (NSString*)windowTitle;
- (void)setSocketPath:(const char*)path;
@end

@implementation AppDelegate
- (void)setWindowTitle:(NSString*)title {
    if (windowTitle) [windowTitle release];
    windowTitle = [title retain];
}

- (NSString*)windowTitle {
    return windowTitle;
}

- (void)setSocketPath:(const char*)path {
    if (sockPath) free((void*)sockPath);
    sockPath = strdup(path);
}

- (void)applicationWillFinishLaunching:(NSNotification *)notification {
    NSWindow *window = [[NSWindow alloc]
        initWithContentRect:NSMakeRect(0, 0, 130, 80)
                  styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable
                    backing:NSBackingStoreBuffered
                      defer:NO];

    windowDelegate = [[WindowDelegate alloc] initWithSocketPath:sockPath];
    [window setDelegate:windowDelegate];
    window.level = NSFloatingWindowLevel;
    window.collectionBehavior = NSWindowCollectionBehaviorFullScreenNone;
    window.styleMask &= ~NSWindowStyleMaskResizable;

    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

    DragView *dragView = [[DragView alloc] initWithFrame:window.contentView.bounds socketPath:sockPath];
    [window.contentView addSubview:dragView];
    [dragView release];

    // Исправленная строка - используем метод windowTitle вместо прямого доступа
    [window setTitle:[self windowTitle] ?: @"dr&Tags"];
    [window center];
    [window makeKeyAndOrderFront:nil];
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
    return YES;
}

- (void)dealloc {
    if (windowTitle) [windowTitle release];
    if (sockPath) free((void*)sockPath);
    [windowDelegate release];
    [super dealloc];
}
@end

// ===== 5. Функция запуска =====
void StartApp(const char* title, const char* sockPath) {
    [NSApplication sharedApplication];
    AppDelegate *delegate = [[AppDelegate alloc] init];

    if (title != NULL) {
        [delegate setWindowTitle:[NSString stringWithUTF8String:title]];
    }

    if (sockPath != NULL) {
        [delegate setSocketPath:sockPath];
    }

    [NSApp setDelegate:delegate];
    [NSApp run];

    [delegate release];
}
*/

import "C"
import (
	"log"
	"os"
	"syscall"
	"unsafe"

	"github.com/xlab/closer"
)

func unixSocketListener(sock string) {
	os.Remove(sock)

	sockFd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Println("Socket error:", err)
		return
	}
	defer syscall.Close(sockFd)

	addr := &syscall.SockaddrUnix{Name: sock}
	if err := syscall.Bind(sockFd, addr); err != nil {
		log.Println("Bind error:", err)
		return
	}

	if err := syscall.Listen(sockFd, 5); err != nil {
		log.Println("Listen error:", err)
		return
	}

	// log.Println("UNIX socket listener started on", sock)

	for {
		fd, _, err := syscall.Accept(sockFd)
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}

		buf := make([]byte, 102400)
		n, err := syscall.Read(fd, buf)
		syscall.Close(fd)
		if err != nil {
			log.Println("Read error:", err)
			continue
		}
		if n == 0 {
			closer.Close()
			return
		}

		paths := string(buf[:n])
		logPaths(paths)
	}
}

func showDroplet(title string) {
	reg, err := os.CreateTemp("", "drt*.sock")
	if err != nil {
		log.Println("Failed to create temp socket:", err)
		return
	}
	sock := reg.Name()
	// log.Println("Using socket:", sock)
	reg.Close()
	os.Remove(sock)

	go unixSocketListener(sock)

	cTitle := C.CString(title)
	cSock := C.CString(sock)
	closer.Bind(func() {
		C.free(unsafe.Pointer(cTitle))
		C.free(unsafe.Pointer(cSock))
		os.Remove(sock)
	})

	C.StartApp(cTitle, cSock)
	// никогда
}
