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

    for (NSURL *url in urls) {
        sendToGo([[url path] UTF8String]);
    }
    return YES;
}
@end

void StartApp() {
    NSWindow *window = [[NSWindow alloc]
        initWithContentRect:NSMakeRect(0, 0, 150, 50)
                  styleMask:NSWindowStyleMaskTitled
                    backing:NSBackingStoreBuffered
                      defer:NO];

    DragView *dragView = [[DragView alloc] initWithFrame:window.contentView.bounds];
    [window.contentView addSubview:dragView];

    [window setTitle:@"Drag and Drop for drTags"];
    [window center];
    [window makeKeyAndOrderFront:nil];

    [[NSApplication sharedApplication] run];
}
*/
import "C"
import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func unixSocketListener() {
	// Удаляем старый сокет (если существует)
	os.Remove("/tmp/dragdrop.sock")

	sock, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		fmt.Println("Socket error:", err)
		return
	}
	defer syscall.Close(sock)

	addr := &syscall.SockaddrUnix{Name: "/tmp/dragdrop.sock"}
	if err := syscall.Bind(sock, addr); err != nil {
		fmt.Println("Bind error:", err)
		return
	}

	if err := syscall.Listen(sock, 5); err != nil {
		fmt.Println("Listen error:", err)
		return
	}

	fmt.Println("UNIX socket listener started")

	// Обработка Ctrl+C для очистки сокета
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		os.Remove("/tmp/dragdrop.sock")
		os.Exit(0)
	}()

	for {
		fd, _, err := syscall.Accept(sock)
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}

		buf := make([]byte, 1024)
		n, err := syscall.Read(fd, buf)
		if err != nil {
			fmt.Println("Read error:", err)
			syscall.Close(fd)
			continue
		}

		path := string(buf[:n])
		fmt.Printf("Processing file: %s\n", path)
		syscall.Close(fd)
	}
}

func main() {
	go unixSocketListener()
	C.StartApp() // Запускаем GUI
}
