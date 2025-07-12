//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework ScriptingBridge
#import <Foundation/Foundation.h>
#import <ScriptingBridge/ScriptingBridge.h>

char** getFinderSelection(int *count) {
    *count = 0;

    @autoreleasepool {
        SBApplication *finder = [SBApplication applicationWithBundleIdentifier:@"com.apple.Finder"];
        if (!finder) return NULL;

        SBElementArray *selection = [finder valueForKey:@"selection"];
        if (!selection) return NULL;

        NSArray *items = [selection get];
        int itemCount = (int)[items count];
        if (itemCount == 0) return NULL;

        char **paths = malloc(sizeof(char*) * itemCount);
        if (!paths) return NULL;

        for (int i = 0; i < itemCount; i++) {
            id item = items[i];
            NSString *fullPath = @"";
            BOOL isDirectory = NO;

            @try {
                // Основной способ - через URL
                if ([item respondsToSelector:@selector(URL)]) {
                    id urlValue = [item valueForKey:@"URL"];
                    if ([urlValue isKindOfClass:[NSString class]]) {
                        NSURL *url = [NSURL URLWithString:urlValue];
                        if (url) {
                            fullPath = [url path];
                            // Проверяем является ли путь директорией
                            NSNumber *isDir;
                            if ([url getResourceValue:&isDir forKey:NSURLIsDirectoryKey error:nil]) {
                                isDirectory = [isDir boolValue];
                            }
                        }
                    }
                }

                // Добавляем / в конце для директорий
                if ([fullPath length] > 0 && isDirectory) {
                    if (![fullPath hasSuffix:@"/"]) {
                        fullPath = [fullPath stringByAppendingString:@"/"];
                    }
                }

                // Резервный способ - через POSIX путь
                if ([fullPath length] == 0 && [item respondsToSelector:@selector(POSIXPath)]) {
                    id posixValue = [item valueForKey:@"POSIXPath"];
                    if ([posixValue isKindOfClass:[NSString class]]) {
                        fullPath = posixValue;
                    }
                }

            } @catch (NSException *e) {
                fullPath = @"[error:invalid_reference]";
            }

            paths[i] = strdup([fullPath UTF8String]);
        }

        *count = itemCount;
        return paths;
    }
}

void freePaths(char **paths, int count) {
    if (!paths) return;
    for (int i = 0; i < count; i++) {
        if (paths[i]) free(paths[i]);
    }
    free(paths);
}
*/
import "C"
import (
	"unsafe"
)

func getSelectedFiles() []string {

	var count C.int
	cPaths := C.getFinderSelection(&count)
	defer func() {
		if cPaths != nil {
			C.freePaths(cPaths, count)
		}
	}()

	if count == 0 || cPaths == nil {
		return nil
	}

	paths := make([]string, int(count))
	for i := 0; i < int(count); i++ {
		ptr := *(**C.char)(unsafe.Pointer(
			uintptr(unsafe.Pointer(cPaths)) + uintptr(i)*unsafe.Sizeof(*cPaths),
		))
		paths[i] = C.GoString(ptr)
	}

	return paths
}
