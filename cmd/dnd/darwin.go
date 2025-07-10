//go:build darwin

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

static void sendToGo(const char* msg) {
    int clientSock = socket(AF_UNIX, SOCK_STREAM, 0);
    if (clientSock == -1) return;

    struct sockaddr_un addr;
    memset(&addr, 0, sizeof(addr));
    addr.sun_family = AF_UNIX;
    strncpy(addr.sun_path, "/tmp/dragdrop.sock", sizeof(addr.sun_path)-1);

    if (connect(clientSock, (struct sockaddr*)&addr, sizeof(addr)) == -1) {
        close(clientSock);
        return;
    }

    write(clientSock, msg, strlen(msg));
    close(clientSock);
}

@interface DragView : NSView <NSDraggingDestination>
@end

@implementation DragView
- (instancetype)initWithFrame:(NSRect)frameRect {
    self = [super initWithFrame:frameRect];
    if (self) {
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

    // Собираем все пути в одну строку с разделителем \n
    NSMutableString *combinedPaths = [NSMutableString string];
    for (NSURL *url in urls) {
        if ([combinedPaths length] > 0) {
            [combinedPaths appendString:@"\n"];
        }
        [combinedPaths appendString:[url path]];
    }

    // Отправляем все пути одной строкой
    if ([combinedPaths length] > 0) {
        sendToGo([combinedPaths UTF8String]);
    }

    return YES;
}
@end

@interface AppDelegate : NSObject <NSApplicationDelegate> {
    NSString* windowTitle;
}
- (void)setWindowTitle:(NSString*)title;
- (NSString*)windowTitle;
@end

@implementation AppDelegate
- (void)setWindowTitle:(NSString*)title {
    windowTitle = title;
}

- (NSString*)windowTitle {
    return windowTitle;
}

- (void)applicationWillFinishLaunching:(NSNotification *)notification {
    NSWindow *window = [[NSWindow alloc]
        initWithContentRect:NSMakeRect(0, 0, 160, 80)
                  styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable
                    backing:NSBackingStoreBuffered
                      defer:NO];

    // Настройки окна
    window.level = NSFloatingWindowLevel;
    window.collectionBehavior = NSWindowCollectionBehaviorFullScreenNone;
    window.styleMask &= ~NSWindowStyleMaskResizable;

    // Отключаем Dock-иконку
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

    DragView *dragView = [[DragView alloc] initWithFrame:window.contentView.bounds];
    [window.contentView addSubview:dragView];

    // Используем переданный заголовок или значение по умолчанию
    [window setTitle:self.windowTitle ?: @"dr&Tags"];
    [window center];
    [window makeKeyAndOrderFront:nil];
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
    return YES;
}
@end

void StartApp(const char* title) {
    [NSApplication sharedApplication];
    AppDelegate *delegate = [[AppDelegate alloc] init];

    if (title != NULL) {
        [delegate setWindowTitle:[NSString stringWithUTF8String:title]];
    }

    [NSApp setDelegate:delegate];
    [NSApp run];
}
*/
import "C"
import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

func unixSocketListener(sock string) {
	sock, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		fmt.Println("Socket error:", err)
		return
	}
	defer syscall.Close(sock)

	addr := &syscall.SockaddrUnix{Name: sock}
	if err := syscall.Bind(sock, addr); err != nil {
		fmt.Println("Bind error:", err)
		return
	}

	if err := syscall.Listen(sock, 5); err != nil {
		fmt.Println("Listen error:", err)
		return
	}

	fmt.Println("UNIX socket listener started")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		os.Remove(sock)
		os.Exit(0)
	}()

	for {
		fd, _, err := syscall.Accept(sock)
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}

		buf := make([]byte, 102400)
		n, err := syscall.Read(fd, buf)
		if err != nil {
			fmt.Println("Read error:", err)
			syscall.Close(fd)
			continue
		}

		paths := string(buf[:n])
		// fmt.Printf("Processing file: %s\n", path)
		logPaths(paths)
		syscall.Close(fd)
	}
}

func showDroplet(title string) {
	reg, err := os.CreateTemp("", drt+"*.sock")
	if err != nil {
		log.Println("~> sock", err)
		return
	}
	sock := reg.Name()
	log.Println("~>", sock)
	reg.Close()
	os.Remove(sock)

	go unixSocketListener(sock)

	cTitle := C.CString(title)
	cSock := C.CString(sock)
	defer func() {
		C.free(unsafe.Pointer(cTitle))
		C.free(unsafe.Pointer(cSock))
		os.Remove(sock)
	}()

	C.StartApp(cTitle, cSock)
}
