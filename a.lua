--[[
 * Resolve Script Name: Export Poster Markers as Still Frames (Universal Version)
 * Author: abakum
 * Licence: GPL v3
 * Version: 2.7 (автоматический рендеринг и обработка)
--]]

local VERSION = "2.7"
local DEFAULT_EXPORT_FOLDER = "c:/tmp/"
--local FORMAT = "png"
--local CODEC = "RGB8"
local FORMAT = "dpx"
local CODEC = "RGB10"

function init()
    print(string.format("=== Экспорт кадров с маркерами (v%s) ===", VERSION))
    
    resolve = Resolve()
    if not resolve then
        print("Ошибка: Не удалось получить объект Resolve")
        return false
    end

    project = resolve:GetProjectManager():GetCurrentProject()
    if not project then
        print("Ошибка: Проект не найден")
        return false
    end
    frameRate = tonumber(project:GetSetting("timelineFrameRate")) or 24
    exported = 0
    renderJobsAdded = false
    outputFolder = ""
    
    return true
end

function AddLeadingZeros(num)
    return string.format("%02d", tonumber(num) or 0)
end

function TimeToTimecode(seconds, fps)
    seconds = tonumber(seconds) or 0
    fps = tonumber(fps) or 24
    
    local h = math.floor(seconds / 3600)
    local m = math.floor((seconds % 3600) / 60)
    local s = math.floor(seconds % 60)
    local f = math.floor((seconds - math.floor(seconds)) * fps)
    
    return string.format("%s:%s:%s:%s", 
        AddLeadingZeros(h), 
        AddLeadingZeros(m), 
        AddLeadingZeros(s), 
        AddLeadingZeros(f))
end

function TimecodeToFrames(timecode, fps)
    if type(timecode) == "number" then return timecode end
    if type(timecode) ~= "string" then return 0 end
    
    local h, m, s, f = timecode:match("(%d+):(%d+):(%d+):(%d+)")
    h, m, s, f = tonumber(h), tonumber(m), tonumber(s), tonumber(f)
    
    return f + s * fps + m * 60 * fps + h * 3600 * fps
end

function GetParentDirectory(filepath)
    if type(filepath) ~= "string" or filepath == "" then return nil end
    
    filepath = filepath:gsub("\\", "/"):gsub("/+$", "")
    local parentDir = filepath:match("^(.*)/[^/]*$") or filepath
    
    if filepath:match("^%a:/") then
        parentDir = parentDir:gsub("/", "\\") .. "\\"
    end
    
    return parentDir
end

function GetCurrentClip()

    local timeline = project:GetCurrentTimeline()
    if not timeline then return nil end
    
    local currentPos = timeline:GetCurrentTimecode()
    local currentFrames = TimecodeToFrames(currentPos, frameRate)
    
    for trackIndex = 1, 3 do
        local clips = timeline:GetItemListInTrack("video", trackIndex)
        if clips then
            for _, clip in ipairs(clips) do
                local start = TimecodeToFrames(clip:GetStart(), frameRate)
                local duration = clip:GetDuration()
                
                if currentFrames >= start and currentFrames <= start + duration - 1 then
                    return clip
                end
            end
        end
    end
    
    return nil
end

function SanitizeFilename(name)
    if not name or type(name) ~= "string" then return "frame" end
    return name:gsub("[\\/:*?\"<>|]", "_")
               :gsub("%s+", " ")
               :gsub("^%s+", "")
               :gsub("%s+$", "")
end

function ExportFrameAsStill(pos, outputPath)
    -- Пробуем современный метод, если доступен и старый метод если клипы не найдены
    local TargetDir = GetParentDirectory(outputPath)
    if type(project.ExportCurrentFrameAsStill) == "fu nction" and DEFAULT_EXPORT_FOLDER ~= TargetDir then
        if project:ExportCurrentFrameAsStill(outputPath) then
            return true
        end
        print("Со вкладки Deliver даже в новой версии вместо ExportCurrentFrameAsStill будет вызван AddRenderJob")
    end
    
    -- Альтернативный метод через рендер-задания
    local timeline = project:GetCurrentTimeline()
    local startFrame = timeline:GetStartFrame()
    
    local renderSettings = {
        MarkIn = startFrame + pos,
        MarkOut = startFrame + pos,
        --TargetDir = TargetDir,
        CustomName = outputPath:match("([^\\/]+)$"):gsub("%..+$", ""),
        ExportVideo = false,
        ExportAudio = false
    }
    
    if not project:SetRenderSettings(renderSettings) then
        print("Ошибка: Не удалось установить настройки рендера")
        return false
    end
    
    if not project:SetCurrentRenderFormatAndCodec(FORMAT, CODEC) then
        local FORMAT = "dpx"
        local CODEC = "RGB10"
        if not project:SetCurrentRenderFormatAndCodec(FORMAT, CODEC) then
            print(string.format("Ошибка: Не удалось установить формат %s и кодек %s", FORMAT, CODEC))
            return false
        end
    end

    if project:AddRenderJob() then
        renderJobsAdded = true
        return true
    end
    return false
end

function ExportMarkedFrames()
    
    local timeline = project:GetCurrentTimeline()
    if not timeline then
        print("Ошибка: Таймлайн не найден")
        return
    end
    
    local markers = timeline:GetMarkers()
    
    if not markers or type(markers) ~= "table" or not next(markers) then
        print("Ошибка: Маркеры не найдены")
        return
    end
    
    -- Получаем путь для экспорта
    local clip = GetCurrentClip()
    outputFolder = DEFAULT_EXPORT_FOLDER
    
    if clip then
        local mediaPoolItem = clip:GetMediaPoolItem()
        if mediaPoolItem then
            local clipPath = mediaPoolItem:GetClipProperty("File Path")
            if clipPath and clipPath ~= "" then
                outputFolder = GetParentDirectory(clipPath)
                print("Найден клип: " .. clipPath)
            end
        end
    end
    
    -- Сортировка маркеров
    local positions = {}
    for pos in pairs(markers) do table.insert(positions, pos) end
    table.sort(positions)
    
    -- Базовое имя файла
    local timelineName = SanitizeFilename(timeline:GetName() or "frame")
    
    -- Экспорт кадров
    for _, pos in ipairs(positions) do
        local marker = markers[pos]
        
        local name = marker.name
        if marker.name == "Marker 1" then
            name = ""
        end

        timeline:SetCurrentTimecode(pos)
        
        local timecode = TimeToTimecode(pos/frameRate, frameRate)
        --local cleanTimecode = timecode:gsub(":", "_")
        local filename = string.format("%s%s", 
            timelineName, name)
        local fullPath = outputFolder .. filename .. "." .. FORMAT
        
        if ExportFrameAsStill(pos, fullPath) then
            print(string.format("Экспортирован %s -> %s", timecode, filename))
            exported = exported + 1
        else
            print("Ошибка экспорта: " .. timecode)
        end
    end
end  

function GetFilesInDirectory(path)
    local files = {}
    local handle
    
    if package.config:sub(1,1) == "\\" then -- Windows
        handle = io.popen('dir /b "'..path..'" 2>nul')
    else -- Linux/Mac
        handle = io.popen('ls -1 "'..path..'" 2>/dev/null')
    end
    
    if handle then
        for file in handle:lines() do
            table.insert(files, file)
        end
        handle:close()
    else
        print("Ошибка чтения директории")
    end
    
    return files
end

function GetFilesInFolder(folderPath)
    local mediaStorage = resolve:GetMediaStorage()
    if not mediaStorage then
        print("Ошибка: Не удалось получить доступ к MediaStorage")
        return {}
    end
    
    local files = mediaStorage:GetFileList(folderPath)
    return files or {}
end

local function rename(src, dst)
    local content = io.open(src, "rb"):read("*a")
    local out = io.open(dst, "wb")
    out:write(content)
    out:close()
    --os.remove(src)
    return true
end

function ProcessRenderResults(outputFolder)
    print("\nОбработка результатов рендеринга...")
    
    -- Переходим на вкладку Deliver
    --resolve:OpenPage("Deliver")
    
    
    -- Запускаем рендеринг
    if project:StartRendering() then
        print("Рендеринг запущен...")
        
        -- Ожидаем завершения рендеринга
        while project:IsRenderingInProgress() do
            os.execute("ping -n 3 127.0.0.1 > nul") -- Задержка 1 секунда
        end
        
        print("Рендеринг завершен!")
        
        -- Обработка файлов
        print(outputFolder)
        local files = GetFilesInFolder(outputFolder)
        if #files == 0 then
            print("Нет файлов для обработки в:", outputFolder)
        else
            for _, file in ipairs(files) do
                local originalPath =  file
                local newName = file:gsub("(_?%d%d%d%d%d%d%d%d)(%.png)$", "%2")  -- Для .png
                                    :gsub("(_?%d%d%d%d%d%d%d%d)(%.dpx)$", "%2")  -- Для .dpx
                
                if file ~= newName then
                    local newPath =  newName
                    local src = originalPath:gsub("\\", "/"):gsub(" ", "\\ ")
                    print(file .. " ~> ")
                    local dst = newPath:gsub("\\", "/"):gsub(" ", "\\ ")
                    print(newName)
                    local ok = rename(file, newName)
                    print(ok)
                    
                    -- Конвертируем DPX в PNG если нужно
                    --if file:match("%.dpx$") then
                    --    local pngPath = newPath:gsub("%.dpx$", ".png")
                    --    ConvertDpxToPng(newPath, pngPath)
                    --    --os.remove(newPath)
                    --end
                end
            end
        end
    else
        print("Ошибка запуска рендеринга")
    end
end

function ConvertDpxToPng(dpxPath, pngPath)
    local fusion = resolve:Fusion()
    if not fusion then
        print("Ошибка: Не удалось получить доступ к Fusion")
        return false
    end
    
    local comp = fusion:NewComp()
    if not comp then
        print("Ошибка: Не удалось получить композицию")
        return false
    end
    
    -- Создаем инструменты для конвертации
    local loader = comp:AddTool("Loader")
    local saver = comp:AddTool("Saver")
    
    loader.Clip = dpxPath
    saver.Clip = pngPath
    saver["FormatConfig.FileFormat"] = "PNG"
    
    -- Соединяем инструменты
    saver:ConnectInput("Input", loader, "Output")
    
    -- Рендерим один кадр
    local success = comp:Render({
        Start = 0,
        End = 0,
        Wait = true
    })
    
    loader:Delete()
    saver:Delete()
    
    return success
end

-- Запуск
if init() then
    project.DeleteAllRenderJobs()

    ExportMarkedFrames()
    
    print(string.format("\n=== Успешно экспортировано: %d кадров ===", exported))
    
    if renderJobsAdded then
        ProcessRenderResults(outputFolder)
    end
end