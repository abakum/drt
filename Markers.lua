--[[
 * Resolve Script Name: Export Poster Markers as Still Frames
 * Author: abakum
 * Licence: GPL v3
 * Version: 2.1 (исправлено для кириллицы)
--]]

local VERSION = "2.1"
local DEFAULT_EXPORT_FOLDER = "c:/tmp/"

function init()
    print(string.format("=== Экспорт кадров с маркерами (v%s) ===", VERSION))
    
    -- Основной объект Resolve
    resolve = Resolve()
    if not resolve then
        print("Ошибка: Не удалось получить объект Resolve")
        return false
    end

    project = resolve:GetProjectManager():GetCurrentProject()
    if not project then
        print("Ошибка: Проект не найден")
        return
    end
    frameRate = tonumber(project:GetSetting("timelineFrameRate")) or 24
    
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
    
    -- Нормализация путей
    filepath = filepath:gsub("\\", "/"):gsub("/+$", "")
    local parentDir = filepath:match("^(.*)/[^/]*$") or filepath
    
    -- Для Windows возвращаем с обратными слешами
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
    
    for trackIndex = 1, 3 do -- Проверяем первые 3 видео трека
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
    -- Заменяем запрещенные символы на подчеркивания
    if not name or type(name) ~= "string" then return "frame" end
    
    -- Сохраняем кириллицу, заменяем только специальные символы
    return name:gsub("[\\/:*?\"<>|]", "_")
               :gsub("%s+", " ")
               :gsub("^%s+", "")
               :gsub("%s+$", "")
end

function ExportMarkedFrames()
    if not init() then return end
    
    
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
    local outputFolder = DEFAULT_EXPORT_FOLDER
    
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
    
    print("Папка для экспорта: " .. outputFolder)
    
    -- Сортировка маркеров
    local positions = {}
    for pos in pairs(markers) do table.insert(positions, pos) end
    table.sort(positions)
    
    -- Базовое имя файла
    local timelineName = SanitizeFilename(timeline:GetName() or "frame")
    
    -- Экспорт кадров
    local exported = 0
    for _, pos in ipairs(positions) do
        local marker = markers[pos]
        
        local name = marker.name
        if marker.name == "Marker 1" then
            name =""
        end

        timeline:SetCurrentTimecode(pos)
        
        local timecode = TimeToTimecode(pos/frameRate, frameRate)
        local cleanTimecode = timecode:gsub(":", "_")
        local filename = string.format("%s%s.png", 
            timelineName,name)
        
        if project:ExportCurrentFrameAsStill(outputFolder .. filename) then
            print(string.format("Экспортирован %s -> %s", timecode, filename))
            exported = exported + 1
        else
            print("Ошибка экспорта: " .. timecode)
        end
    end
    
    print(string.format("\nГотово! Успешно экспортировано: %d кадров", exported))
end

-- Запуск
ExportMarkedFrames()