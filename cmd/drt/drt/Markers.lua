-- Объявление глобальных переменных
resolve = Resolve()
ProjectManager = resolve:GetProjectManager()
Project = ProjectManager:GetCurrentProject()
MediaPool = Project:GetMediaPool()
RootFolder = MediaPool:GetRootFolder()
Clips = RootFolder:GetClipList()
OutputFolder = ""
FrameRate = 0.0
Exported = 0
Total = 0
RenderJobsAdded = false

-- Константы
local FORMAT = "png"
local CODEC = "RGB8"
local FORMATb = "tif"
local CODECb = "RGB8LZW"

-- Функция преобразования времени в таймкод
local function timeToTimecode(seconds, fps)
    return string.format("%02d:%02d:%02d:%02d",
        math.floor(seconds / 3600),
        math.floor(math.fmod(seconds, 3600) / 60),
        math.floor(math.fmod(seconds, 60)),
        math.floor(math.fmod(seconds, 1) * fps))
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
    if type(Project.ExportCurrentFrameAsStill) == "function" then
        if Project:ExportCurrentFrameAsStill(outputPath) then
            return true
        end
        if resolve:GetCurrentPage() == "deliver" then
            print("С панели Deliver даже в новой версии вместо ExportCurrentFrameAsStill будет вызван AddRenderJob")
        end
    end

    -- План B через Deliver
    local timeline = Project:GetCurrentTimeline()
    local startFrame = timeline:GetStartFrame()

    local renderSettings = {
        MarkIn = startFrame + pos,
        MarkOut = startFrame + pos,
        CustomName = string.gsub(outputPath:match("([^/\\]+)$"), "%..+$", ""),
        TargetDir = OutputFolder,
        ExportVideo = false,
        ExportAudio = false
    }

    if not Project:SetRenderSettings(renderSettings) then
        print("Не удалось установить настройки рендера")
        return false
    end

    if not Project:SetCurrentRenderFormatAndCodec(FORMAT, CODEC) then
        if not Project:SetCurrentRenderFormatAndCodec(FORMATb, CODECb) then
            print(string.format("Не удалось установить формат %s и кодек %s", FORMATb, CODECb))
            return false
        end
    end

    local jobId = Project:AddRenderJob()
    if jobId ~= "" then
        RenderJobsAdded =true
    end
    return true
end

-- Функция экспорта помеченных кадров
local function exportMarkedFrames()
    local timeline = Project:GetCurrentTimeline()
    local timelineName = sanitizeFilename(timeline:GetName())
    local markers = timeline:GetMarkers()
    
    if not markers or type(markers) ~= "table" or not next(markers) then
        print("Не найдены маркеры в " .. timelineName)
        return
    end

    -- Сортировка маркеров
    local positions = {}
    for pos, _ in pairs(markers) do
        table.insert(positions, pos)
    end
    table.sort(positions)

    -- Экспорт кадров
    for _, pos in ipairs(positions) do
        local marker = markers[pos]
        local name = marker.name
        if name == "Marker 1" then
            name = ""
        end

        local timecode = timeToTimecode(pos / FrameRate, FrameRate)
        timeline:SetCurrentTimecode(timecode)

        local filename = timelineName .. name
        local fullPath = OutputFolder .. "/" .. filename .. "." .. FORMAT

        if exportFrameAsStill(pos, fullPath) then
            print(string.format("Экспортирован %s -> %s", timecode, filename))
            Exported = Exported + 1
        else
            print("Ошибка экспорта " .. timecode)
        end
        Total = Total + 1
    end
end

-- Функция экспорта всех таймлайнов
local function exportAllTimelines()
    local timelineCount = Project:GetTimelineCount()

    -- Сохраняем оригинальный таймлайн
    local originalTimeline = Project:GetCurrentTimeline()

    -- Обрабатываем все таймлайны
    for i = 1, timelineCount do
        local timeline = Project:GetTimelineByIndex(i)
        Project:SetCurrentTimeline(timeline)
        exportMarkedFrames()
    end

    -- Восстанавливаем оригинальный таймлайн
    if timelineCount > 1 then
        Project:SetCurrentTimeline(originalTimeline)
    end
end

-- Функция получения корневой папки и подсчета маркеров
local function root()
    
    for _, clip in ipairs(Clips) do
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

    if not Project then
        print("Проект не найден")
        return
    end
    
    OutputFolder = root()
    if OutputFolder == "" then
        print("Пустой медиапул")
        return
    end
    
    local fpsStr = Project:GetSetting("timelineFrameRate")
    FrameRate = tonumber(fpsStr) or 24.0

    Project:DeleteAllRenderJobs()
    if resolve:GetCurrentPage() == "edit" then
		print("С панели Edit экспорт кадров с маркерами для одного таймлайна")
        exportMarkedFrames()
    else
        exportAllTimelines()
    end


    print(string.format("=== Экспортировано %d из %d ===", Exported, Total))
    if RenderJobsAdded then
        Project:StartRendering()
    end
end

-- Запуск программы
main()