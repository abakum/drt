//go:build ignore

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
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"unsafe"

	"github.com/adrg/xdg"
	"github.com/google/shlex"
	"github.com/xlab/closer"
)

func unixSocketListener() {
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
	go unixSocketListener()

	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))

	C.StartApp(cTitle)
}

const (
	applescript = "main.applescript"
	Resources   = "Resources"
)

//go:embed drTags.app
var app embed.FS

//go:embed drTags.workflow
var workflow embed.FS

var (
	met = map[string]string{
		dotCSV:  "public.comma-separated-values-text",
		dotMOV:  "com.apple.quicktime-movie",
		dotMP4:  "public.mpeg-4",
		dotFLAC: "org.xiph.flac",
		dotMP3:  "public.mp3",
	}
	DRS = []string{
		filepath.Join(filepath.Dir(xdg.DataHome),
			"Containers",
			"com.blackmagic-design.DaVinciResolveLite",
			"Data",
			"Library",
			"Application Support",
		),
		filepath.Join(xdg.DataHome, "Blackmagic Design", "DaVinci Resolve"),
		filepath.Join(xdg.DataDirs[0], "Blackmagic Design", "DaVinci Resolve"),
	}
)

func onMain() {
	if strings.ToLower(args0) != droplet {
		return
	}
	MacOS := filepath.Dir(os.Args[0])
	Contents := filepath.Dir(MacOS)
	scriptName := filepath.Join(Contents, Resources, applescript)
	script, err := os.ReadFile(scriptName)
	if err == nil {
		cmd := exec.Command("osascript", "-e", string(script))
		// log.Println(string(script))
		output, err := cmd.CombinedOutput()
		log.Println(cmd, err)
		if err != nil {
			log.Println(string(output))
		}
	} else {
		log.Println("<~", scriptName, err)
	}
	closer.Close()
}

// https://github.com/RichardBronosky/AppleScript-droplet
func install(oldname string, lnks ...string) {
	adr, link := lnks[0], lnks[1]
	// /dest/drTags.app dir/drTags
	servicePath := filepath.Join(filepath.Dir(xdg.DataHome), "Services", drTags+".workflow") // dir
	if oldname == "" {
		//uninstall
		log.Println(adr, "~> /dev/null", os.RemoveAll(adr))
		log.Println(servicePath, "~> /dev/null", os.RemoveAll(servicePath))
		log.Println(link, "~> /dev/null", os.Remove(link))
		return
	}
	ln(oldname, link, true, false)

	// Грязный результат fn
	var script []byte

	// Грязные параметры для fn
	_, err := exec.LookPath(drTags)
	replace := err != nil
	dest := filepath.Dir(servicePath)
	efs := workflow

	// Walk through the embedded directory and copy files/dirs.
	fn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, path)
		// log.Println(path, destPath, d.Name(), "path,destPath,d.Name()")

		if d.IsDir() {
			// Create destination directory if it doesn't exist.
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				err = os.MkdirAll(destPath, 0755)
				if err != nil {
					log.Println("Error creating directory:", err)
					return err
				}
			}
			return nil
		}

		if filepath.Base(path) == applescript {
			script, err = efs.ReadFile(path)
			log.Println("<~", path, err)
			if err != nil {
				return err
			}
			if replace {
				script = bytes.Replace(script, []byte(`"`+drTags+`"`), []byte(`"`+filepath.Join(dir, drTags)+`"`), 1)
				err = os.WriteFile(destPath, script, 0644)
				log.Println("~>", path, err)
				if err != nil {
					return err
				}
				return nil
			}
		}

		// Copy file.
		srcFile, err := efs.Open(path)
		if err != nil {
			log.Println("Error opening embedded file:", err)
			return err
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			log.Println("Error creating destination file:", err)
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, srcFile)
		log.Println(path, "~>", destPath, err)
		if err != nil {
			return err
		}
		return nil
	}

	fs.WalkDir(efs, ".", fn)
	pbs(servicePath, string(script))

	dest = filepath.Dir(adr)
	efs = app
	fs.WalkDir(efs, ".", fn)
	Contents := filepath.Join(adr, "Contents")
	MacOS := filepath.Join(Contents, "MacOS")
	drtlet := filepath.Join(MacOS, droplet)
	os.MkdirAll(MacOS, 0755)
	ln(oldname, drtlet, true, false)
	log.Println("chmod +x", drtlet, os.Chmod(drtlet, 0755))

}

func evtp() {
}

func SplitCommandLine(command string) ([]string, error) {
	return shlex.Split(command)
}

func mkLink(oldname, newname string, link, hard bool) (err error) {
	return ln(oldname, newname, link, hard)
}

func pbs(servicePath, workflowScript string) {
	documentFile := filepath.Join(servicePath, "Contents", "document.wflow")

	// Чтение файла с логгированием
	documentBytes, err := os.ReadFile(documentFile)
	log.Println("<~", documentFile, err)
	if err != nil {
		return
	}

	documentContent := string(documentBytes)
	log.Printf(documentContent)
	log.Printf(workflowScript)

	// Улучшенное регулярное выражение для поиска блока ActionParameters
	re := regexp.MustCompile(`(?s)<key>ActionParameters</key>\s*<dict>\s*<key>source</key>\s*<string>(.*?)</string>`)

	// Заменяем содержимое CDATA
	if re.MatchString(documentContent) {
		// Заменяем существующий скрипт
		documentContent = re.ReplaceAllString(documentContent,
			`<key>ActionParameters</key>
                <dict>
                    <key>source</key>
                    <string><![CDATA[`+workflowScript+`]]></string>`)
	} else {
		// Если не нашли стандартную структуру, вставляем в более общем месте
		re = regexp.MustCompile(`(?s)<key>source</key>\s*<string>(.*?)</string>`)
		if re.MatchString(documentContent) {
			documentContent = re.ReplaceAllString(documentContent,
				`<key>source</key>
                    <string><![CDATA[`+workflowScript+`]]></string>`)
		} else {
			log.Println("Не удалось найти место для вставки скрипта в document.wflow")
			return
		}
	}

	// Запись обратно с логгированием
	err = os.WriteFile(documentFile, []byte(documentContent), 0644)
	log.Println("~>", documentFile, err)
	if err != nil {
		return
	}

	// Обновляем сервисы
	cmd := exec.Command("/System/Library/CoreServices/pbs", "-flush")
	err = cmd.Start()
	log.Println(cmd, err)
	if err != nil {
		return
	}
	cmd.Wait()
}

// tell application "Finder"
//     activate
//     -- Снимаем выделение во всех окнах
//     repeat with win in windows
//         set selection of win to {}
//     end repeat
// end tell

//osascript -e 'on run {input, parameters} ... end run' "/path/to/file1" "/path/to/file2"
