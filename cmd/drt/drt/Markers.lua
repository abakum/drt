-- Объявление глобальных переменных
resolve = Resolve()
projectManager = resolve:GetProjectManager()
project = projectManager:GetCurrentProject()
mediaPool = project:GetMediaPool()
rootFolder = mediaPool:GetRootFolder()
clips = rootFolder:GetClipList()
jobs = {}
outputFolder = ""
frameRate = 0.0
exported = 0
total = 0
renderJobsAdded = false

-- Константы
local FORMAT = "png"
local CODEC = "RGB8"
local FORMATb = "dpx"
local CODECb = "RGB10"

-- Функция преобразования времени в таймкод
local function timeToTimecode(seconds, fps)
    return string.format("%02d:%02d:%02d:%02d",
        math.floor(seconds / 3600),
        math.floor(math.fmod(seconds, 3600) / 60),
        math.floor(math.fmod(seconds, 60)),
        math.floor(math.fmod(seconds, 1) * fps))
end

-- Функция преобразования таймкода в кадры
local function timecodeToFrames(timecode, fps)
    if type(timecode) == "number" then
        return math.floor(timecode)
    elseif type(timecode) == "string" then
        local parts = {}
        for part in string.gmatch(timecode, "([^:]+)") do
            table.insert(parts, part)
        end
        if #parts ~= 4 then return 0 end
        
        local h = tonumber(parts[1]) or 0
        local m = tonumber(parts[2]) or 0
        local s = tonumber(parts[3]) or 0
        local f = tonumber(parts[4]) or 0
        
        return f + s * fps + m * 60 * fps + h * 3600 * fps
    else
        return 0
    end
end

-- Функция очистки имени файла
local function sanitizeFilename(name)
    if name == "" then
        return "frame"
    end
    
    -- Замена недопустимых символов
    name = string.gsub(name, '[\\/:*?"<>|]', '_')
    -- Удаление лишних пробелов
    name = string.gsub(name, '%s+', ' ')
    return string.gsub(name, '^%s*(.-)%s*$', '%1')
end

-- Функция экспорта кадра
local function exportFrameAsStill(pos, outputPath)
    -- Попробуем новый метод
    if project:ExportCurrentFrameAsStill(outputPath) then
        return true
    end

	if resolve:GetCurrentPage() == "deliver" then
		print("С панели Deliver даже в новой версии вместо ExportCurrentFrameAsStill будет вызван AddRenderJob")
    end

    -- План B через Deliver
    local timeline = project:GetCurrentTimeline()
    local startFrame = timeline:GetStartFrame()

    local renderSettings = {
        MarkIn = startFrame + pos,
        MarkOut = startFrame + pos,
        CustomName = string.gsub(outputPath:match("([^/\\]+)$"), "%..+$", ""),
        TargetDir = outputFolder,
        ExportVideo = false,
        ExportAudio = false
    }

    if not project:SetRenderSettings(renderSettings) then
        print("Не удалось установить настройки рендера")
        return false
    end

    if not project:SetCurrentRenderFormatAndCodec(FORMAT, CODEC) then
        if not project:SetCurrentRenderFormatAndCodec(FORMATb, CODECb) then
            print(string.format("Не удалось установить формат %s и кодек %s", FORMATb, CODECb))
            return false
        end
    end

    local jobId = project:AddRenderJob()
    table.insert(jobs, jobId)
    renderJobsAdded = true
    return true
end

-- Функция экспорта помеченных кадров
local function exportMarkedFrames()
    local timeline = project:GetCurrentTimeline()
    local markers = timeline:GetMarkers()
    
    if not markers or type(markers) ~= "table" or not next(markers) then
        print("Маркеры не найдены")
        return
    end

    -- Сортировка маркеров
    local positions = {}
    for pos, _ in pairs(markers) do
        table.insert(positions, pos)
    end
    table.sort(positions)

    -- Базовое имя файла
    local timelineName = sanitizeFilename(timeline:GetName())

    -- Экспорт кадров
    for _, pos in ipairs(positions) do
        local marker = markers[pos]
        local name = marker.name
        if name == "Marker 1" then
            name = ""
        end

        local timecode = timeToTimecode(pos / frameRate, frameRate)
        timeline:SetCurrentTimecode(timecode)

        local filename = timelineName .. name
        local fullPath = outputFolder .. "/" .. filename .. "." .. FORMAT

        if exportFrameAsStill(pos, fullPath) then
            print(string.format("Экспортирован %s -> %s", timecode, filename))
            exported = exported + 1
        else
            print("Ошибка экспорта " .. timecode)
        end
        total = total + 1
    end
end

-- Функция экспорта всех таймлайнов
local function exportAllTimelines()
    local timelineCount = project:GetTimelineCount()

    -- Сохраняем оригинальный таймлайн
    local originalTimeline = project:GetCurrentTimeline()

    -- Обрабатываем все таймлайны
    for i = 1, timelineCount do
        local timeline = project:GetTimelineByIndex(i)
        print(string.format("\n--- Таймлайн: %s ---", timeline:GetName()))
        project:SetCurrentTimeline(timeline)
        exportMarkedFrames()
    end

    -- Восстанавливаем оригинальный таймлайн
    if timelineCount > 1 then
        project:SetCurrentTimeline(originalTimeline)
    end
end

-- Функция получения корневой папки и подсчета маркеров
local function root()
    
    for _, clip in ipairs(clips) do
        local filePath = clip:GetClipProperty("File Path")
        if filePath ~= "" then
            -- Допустим исходные и результирующие медиафайлы в одном каталоге
            return string.match(filePath, "(.*[/\\])") or ""
        end
    end
    
    return ""
end

-- Основная функция
local function main()
    print("=== Экспорт кадров с маркерами ===")

    if not project then
        print("Проект не найден")
        return
    end
    
    outputFolder = root()
    if outputFolder == "" then
        print("Пустой медиапул")
        return
    end
    
    local fpsStr = project:GetSetting("timelineFrameRate")
    frameRate = tonumber(fpsStr) or 24.0

    project:DeleteAllRenderJobs()
    exportAllTimelines()

    print(string.format("=== Экспортировано %d из %d ===", exported, total))
    if renderJobsAdded then
        project:StartRendering()
    end
end

-- Запуск программы
main()