--[[
 * Resolve Script Name: Export Poster Markers as Still Frames (Universal Version)
 * Author: abakum
 * Licence: GPL v3
 * Version: 2.6 (универсальный экспорт)
--]]

local VERSION = "2.6"
local DEFAULT_EXPORT_FOLDER = "c:/tmp/"
local FORMAT = "png"
local CODEC = "RGB8"

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
    render=""

    
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
    if type(project.ExportCurrentFrameAsStill) == "function" and DEFAULT_EXPORT_FOLDER ~= TargetDir then
        if project:ExportCurrentFrameAsStill(outputPath) then
            return true
        end
        print("Со вкладки Deliver даже в новой версии вместо ExportCurrentFrameAsStill будет вызван AddRenderJob")
    end
    
    -- Альтернативный метод через рендер-задания
    local startFrame = timeline:GetStartFrame()
    
    local renderSettings = {
        MarkIn = startFrame + pos,
        MarkOut = startFrame + pos,
        --TargetDir = GetParentDirectory(outputPath),
        CustomName = outputPath:match("([^\\/]+)$"):gsub("%..+$", ""),
        ExportVideo = false,
        ExportAudio = false
    }
    
    if not project:SetRenderSettings(renderSettings) then
        print("Ошибка: Не удалось установить настройки рендера")
        return false
    end
    
    if not project:SetCurrentRenderFormatAndCodec(FORMAT, CODEC) then
        -- Для старых версий
        local FORMAT = "dpx"
        local CODEC = "RGB10"
        if not project:SetCurrentRenderFormatAndCodec(FORMAT, CODEC) then
            print(string.format("Ошибка: Не удалось установить формат %s и кодек %s", FORMAT, CODEC))
            return false
        end
    end

    local result =project:AddRenderJob()
    if result then
        render="Нажмите кнопку Render All на вкладке Deliver"
    end
    return result
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

-- Запуск
if init() then
    ExportMarkedFrames()
    
    print(string.format("\n=== Успешно экспортировано: %d кадров ===", exported))
    print(render)
end