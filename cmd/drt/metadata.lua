resolve = Resolve()
local projectManager = resolve:GetProjectManager()
local project = projectManager:GetCurrentProject()

-- Проверка проекта
if not project then
    print("Ошибка: Проект не открыт.")
    return
end

-- Настройки экспорта
local exportFolder = "c:/tmp/"
--os.execute('mkdir "'..exportFolder..'" 2>nul')

-- Получаем медиапул
local mediaPool = project:GetMediaPool()
if not mediaPool then
    print("Ошибка: Не удалось получить медиапул.")
    return
end

-- Функция для безопасного имени файла (сохраняет кириллицу)
local function makeSafeFilename(name)
    return name:gsub("[\\/:*?\"<>|]", "_")
end

-- Функция экспорта метаданных
local function exportTimelineMetadata(timeline, folderPath)
    local timelineName = timeline:GetName() or "Без названия"
    local fileName = makeSafeFilename(timelineName)..".csv"
    local filePath = folderPath..fileName
    
    -- Получаем все клипы таймлайна
    local timelineClips = {}
    for trackIdx = 1, 3 do  -- Проверяем первые 3 видео-трека
        local items = timeline:GetItemListInTrack("video", trackIdx) or {}
        for _, item in ipairs(items) do
            local clip = item:GetMediaPoolItem()
            if clip then table.insert(timelineClips, clip) end
        end
    end
    
    if #timelineClips == 0 then
        print("Нет клипов в таймлайне: "..timelineName)
        return false
    end
    
    -- Экспортируем метаданные
    if mediaPool:ExportMetadata(filePath, timelineClips) then
        print("Создан файл: "..fileName)
        return true
    else
        print("Ошибка экспорта: "..timelineName)
        return false
    end
end

-- Основной процесс
print("\nНачало экспорта метаданных...")
local successCount = 0

for i = 1, project:GetTimelineCount() do
    local timeline = project:GetTimelineByIndex(i)
    if timeline and exportTimelineMetadata(timeline, exportFolder) then
        successCount = successCount + 1
    end
end

-- Итоговый отчет
print("\nРезультат:")
print("Успешно экспортировано таймлайнов: "..successCount)
print("Файлы сохранены в: "..exportFolder)
print("Готово!")