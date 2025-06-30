#osadecompile main.scpt > main.applescript
#osacompile -l AppleScript -d main.scpt > main.applescript
#osacompile -o main.scpt main.applescript
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
  set file_list to file_list & " " & quoted form of POSIX path of file_path
 end repeat
 
 if file_list is not "" then
  set the_command to the_command & " " & file_list
 end if
 
 tell application "Terminal"
  set newTab to do script the_command
  activate
 end tell
end process