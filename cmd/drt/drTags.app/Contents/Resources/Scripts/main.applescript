-- cd /Applications/drTags.app/Contents/Resources/Scripts/
-- osadecompile main.scpt > main.applescript
-- osacompile -l AppleScript -d main.scpt > main.applescript
-- osacompile -o main.scpt main.applescript

--https://developer.apple.com/library/archive/documentation/LanguagesUtilities/Conceptual/MacAutomationScriptingGuide/ProcessDroppedFilesandFolders.html#//apple_ref/doc/uid/TP40016239-CH53-SW1
property theExtensionsToProcess : {
    "csv",
    "mov",
    "mp4",
    "flac",
    "mp3",
}
property theTypeIdentifiersToProcess : {
    "public.comma-separated-values-text",
    "com.apple.quicktime-movie",
    "public.mpeg-4",
    "org.xiph.flac",
    "public.mp3",
    "public.folder",
}

on run
 process({})
end run

on open dropped_files
 process(dropped_files)
end open

on process(dropped_files)
 set file_list to ""
 set the_command to "drTags"
 
 repeat with file_path in dropped_files
        tell application "System Events"
            set theExtension to name extension of file_path
            set theTypeIdentifier to type identifier of file_path
        end tell
        if ((theExtensionsToProcess contains theExtension) or (theTypeIdentifiersToProcess contains theTypeIdentifier)) then
          set file_list to file_list & " " & quoted form of POSIX path of file_path
        end if
--   set file_list to file_list & " " & quoted form of POSIX path of file_path
 end repeat
 
 if file_list is not "" then
  set the_command to the_command & " " & file_list
 end if
 
 tell application "Terminal"
  set newTab to do script the_command
  activate
 end tell
end process