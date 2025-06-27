function iif(condition, true_val, false_val)
    if condition then
        return true_val
    else
        return false_val
    end
end

bmd=require("bmd")
ffi=require("ffi")
bit=require("bit")

resolve = Resolve()
local projectManager = assert(resolve:GetProjectManager(),"GetProjectManager")

Project = assert(projectManager:GetCurrentProject(),"GetCurrentProject")
MediaPool = assert(Project:GetMediaPool(),"GetMediaPool")
RootFolder = assert(MediaPool:GetRootFolder(),"GetRootFolder")
Clips =  assert(RootFolder:GetClipList(),"GetClipList")
-- MediaStorage = assert(resolve:GetMediaStorage(),"GetMediaStorage")
Windows = package.config:sub(1, 1) == "\\"
Exported = 0

local FORMAT = "png"
local CODEC = "RGB8"
local FORMATb = "tif"
local CODECb = "RGB8LZW"
local TL ="/tl/"

local function getDir(path)
    -- Replace all backslashes with forward slashes for consistency
    path = path:gsub("\\", "/")
    -- Extract everything up to the last slash
    return path:match("(.*)/") or "."
end


local function getBase(path)
    -- Normalize slashes (optional)
    path = path:gsub("\\", "/")
    -- Extract everything after the last slash
    return path:match("([^/]+)$")
end

-- Функция для получения корневой папки и списка таймлайнов
local function rootTimeLines()
    MediaPool:SetCurrentFolder(RootFolder)
    local dir = ""
    local timeLines = {}
    -- local mInOut = {}
    
    for _, clip in ipairs(Clips) do
        local filePath = clip:GetClipProperty("File Path")
        if filePath == "" then
            -- Это таймлайн
			timeLines[clip:GetName()]=clip
        else
            -- В моих проектах все клипы в одном каталоге
            dir = getDir(filePath)
        end
    end
    
    return dir, timeLines --, mInOut
end

-- Функция для очистки имени файла
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

local function exportFrameAsStill(pos, outputPath, timeline, timecode, tlClip, frame_rate)
    -- Попробуем новый метод
	local CustomName=getBase(outputPath)
    -- dump({
    -- pos=pos,
    -- outputPath=outputPath,
    -- timeline=timeline,
    -- timecode=timecode,
    -- tlClip=tlClip,
    -- frame_rate=frame_rate,
	-- CustomName=CustomName,
    -- })
    local ext=FORMAT
    if type(Project.ExportCurrentFrameAsStill) == "function" then
        local ctc=timeline:GetCurrentTimecode(timecode)
        timeline:SetCurrentTimecode(timecode)
        local ok=Project:ExportCurrentFrameAsStill(outputPath.."."..ext)
        timeline:SetCurrentTimecode(ctc)
        if ok then
            return true
        end
        if Page == "deliver" then
            print("С панели Deliver даже в новой версии вместо ExportCurrentFrameAsStill будет вызван AddRenderJob")
        end
    end

    -- План B через Deliver
	-- Запомним
	local In=tlClip:GetClipProperty("In")
	local Out=tlClip:GetClipProperty("Out")
	local output=getDir(outputPath)
	local preset=getBase(output)

	if Project:SaveAsNewRenderPreset(preset) then
		print("Шаблон экспорта ~> "..preset)
		print("Убедись что на странице `File` выбран режим `Filename uses`=`Custom name` и `Custom name`=`%Timeline Name`")
	end
	if type(resolve.ExportRenderPreset) == "function" then
		local presetFile=output.."/"..preset..".xml"
		if not bmd.fileexists(presetFile) and resolve:ExportRenderPreset(preset, output) then
			print("Шаблон экспорта ~> "..presetFile)
		end
	end

    if not Project:SetCurrentRenderFormatAndCodec(ext, CODEC) then
		ext=FORMATb
        if not Project:SetCurrentRenderFormatAndCodec(ext, CODECb) then
            print(string.format("Не удалось установить формат %s и кодек %s", FORMATb, CODECb))
			-- Вспомним
            return false
        end
    end

	local file=outputPath.."."..ext

	local renderSettings = {
        MarkIn = pos,
        MarkOut = pos,
        TargetDir =output,
        CustomName = CustomName,
    }

    if not Project:SetRenderSettings(renderSettings) then
        print("Не удалось установить настройки рендера")
        return false
    end


	Project:SetCurrentRenderMode(1)
    local jobId = Project:AddRenderJob()

	local ok=jobId ~= ""
    if jobId ~= "" then
		ok=Project:StartRendering(jobId)
		if ok then
			ok=false
			for i=1, 10 do
				bmd.wait(1) --Stop the Script to a set amount of seconds
				if not Project:IsRenderingInProgress() then
					ok=true
					break
				end
			end
			if ok then
				Project:DeleteRenderJob(jobId)
			else
				Project:StopRendering()
			end
		end
     end

	 -- Вспомним
	if tlClip:SetClipProperty("In", In) and tlClip:SetClipProperty("Out", Out) then
 	-- Работает в 19
	else
 	-- Работает в 17 
		renderSettings.MarkIn=luaresolve:frame_from_timecode(In, frame_rate)
		renderSettings.MarkOut=luaresolve:frame_from_timecode(Out, frame_rate)
		Project:SetRenderSettings(renderSettings)
	end

	Project:LoadRenderPreset(preset)


	if ok then
		local paths=bmd.readdir(outputPath.."*."..ext)
		if paths and next(paths) then
			for _, path in ipairs(paths) do
				local newname=TrimFrame(path.Name, ext)
				if newname==getBase(file) then
					local oldpath=paths.Parent..path.Name
					local newpath=oldpath:gsub(path.Name, newname)
					-- dump({oldpath=oldpath,newpath=newpath})
					if rename(oldpath, newpath) then
						print(path.Name.." ~> "..newname)
						break
					end
				end
			end
		end
	end

	return ok
end

local function createTempFile(suffix)
    suffix = suffix or ""
    
    -- Создаем уникальное имя файла
    local tempname = os.tmpname()
    -- Формируем полное имя
    local filename = tempname .. suffix
    
    -- Создаем файл (возвращаем файловый объект и имя файла)
    local file, err = io.open(filename, "w+")
    if not file then
        return nil, "Failed to create temp file: " .. (err or "unknown error")
    end
    
    return {
        file = file,
        name = filename,
        close = function(self)
            if self.file then
                os.remove(self.name)
            end
        end
    }
end

function HasNonAscii(text)
	if type(text) ~= "string" then return false end
	return text:match("[\128-\255]") ~= nil
end

function Quote(s)
    -- Проверяем, что аргумент - строка
    if type(s) ~= "string" then return s end
    
    -- Если есть пробелы и строка ещё не в кавычках
    if s:find("%s") and not (s:sub(1,1) == '"' and s:sub(-1) == '"') then
        return '"' .. s .. '"'
    end
    return s
end

local function renameExecute(func, ...)
    local args = {}
	local needsTempFile = false
	for _,arg in ipairs({...}) do
		needsTempFile=needsTempFile or HasNonAscii(arg)
		args[#args+1]=Quote(arg)
	end
    needsTempFile=needsTempFile and Windows

    if not needsTempFile then
        -- Прямой вызов для простых случаев
        if func == os.rename then
            return os.rename(...)
        else
            local success, exit_type, exit_code = os.execute(...)
            if not success then
                return false, exit_type.." "..tostring(exit_code)
            end
            return true
        end
    end

    -- Создание временного BAT-файла с использованием createTempFile
    local temp, err = createTempFile(".bat")
    if not temp then
        return false, "Failed to create temp file: "..err
    end

    -- Записываем команды в BAT-файл
	local command="@chcp 65001\n"
    if func == os.rename then
		command=command.."move /y "
    end
	command=command..table.concat(args, " ").."\n"
	print(command)
	temp.file:write(command)
    temp.file:close()

    -- Выполнение с обработкой результата
    local success, exit_type, exit_code = os.execute(Quote(temp.name))
    
    -- Автоматическое удаление временного файла через метод close
    temp:close()

    if func == os.rename then
        return success
    else
        if success then
            return true
        else
            return false, exit_type.." "..tostring(exit_code)
        end
    end
end

-- Обертки с четкой сигнатурой возврата
function rename(oldName, newName)
    return renameExecute(os.rename, oldName, newName)  -- всегда возвращает bool
end

function ffmpeg(oldpath, newpath)
    return renameExecute(os.execute, "ffmpeg", "-i", oldpath, newpath)
end


-- Основная функция
local function main()
    print("=== Экспорт метаданных ===")

    -- Получаем корневую папку и таймлайны как mediaPoolItem
    local output, tlClips= rootTimeLines()
    assert(output ~= "","Пустой медиапул")
    assert(next(tlClips) ,"Нет таймлайнов")
    bmd.createdir(output..TL)
    
    -- Таблица для хранения клипов по таймлайнам и mediaId
    local tlm = {} -- структура: tlm[timelineName][mediaId] = mediaPoolItem
    
    -- Получаем количество таймлайнов в проекте
    local tlc = Project:GetTimelineCount()
    local ctl = Project:GetCurrentTimeline()
    local ctlName = ctl:GetName()

    local all = false
	Page=resolve:GetCurrentPage()
    if Page ~= "media" then
		print("С панелей кроме Media только "..ctlName)
    else
		print("С панели Media для всех таймлайнов")
        all=true
    end

    
    -- Перебираем все таймлайны
    for i = 1, tlc do
        local timeline = Project:GetTimelineByIndex(i)
        local tln = timeline:GetName()
        if all or ctlName==tln then

            local frame_rate = luaresolve.frame_rates:get_decimal(timeline:GetSetting("timelineFrameRate"))
            local drop_frame = timeline:GetSetting("timelineDropFrameTimecode") == "1"
            local name = sanitizeFilename(tln)
            local file = output .. TL .. name .. ".drt"
            if not all then
                -- Экспортируем в .drt
                timeline:Export(file, resolve.EXPORT_DRT, resolve.EXPORT_NONE)
            end
            file = output .. TL .. name .. ".srt"

            local markers = timeline:GetMarkers()
            local marker_frames = {}
            local marker_table = {}

            -- Add marker frames to the marker_frames table
            for marker_frame, _ in pairs(markers) do
                marker_frames[#marker_frames+1] = marker_frame
            end
                
            -- Sort the table
            table.sort(marker_frames)

            local count = 0
            local color="Blue"
			local timeline_start = timeline:GetStartFrame()
			local In=tlClips[tln]:GetClipProperty("In")
			local Inf=luaresolve:frame_from_timecode(In, frame_rate)

            for _, frame in ipairs(marker_frames) do
				local recordF = math.max(0,frame-Inf)
				local sourceF = timeline_start+frame
                local marker = markers[frame]

                if all or color == marker.color then
                    count = count + 1
                    
                    if marker.note == "" or marker.duration==1 then
                        -- Без описания или короткое значит картинка
                        local fullPath = output .. "/" .. name
                        if marker.name ~= "Marker 1" then
                            fullPath = output .. "/" .. name .. " " .. sanitizeFilename(marker.name)
                        end
                       Project:SetCurrentTimeline(timeline)
                       exportFrameAsStill(sourceF,
					    fullPath, 
						timeline,
						luaresolve:timecode_from_frame(sourceF, frame_rate, drop_frame),
						tlClips[tln],
						frame_rate)
                    else
						local data =
						{
							index = count,
							frame = recordF,
							name = marker.name,
							start_tc = luaresolve:timecode_from_frame(recordF, frame_rate, drop_frame),
							end_tc = luaresolve:timecode_from_frame(recordF + frame + marker.duration, frame_rate, drop_frame),
							duration_tc = luaresolve:timecode_from_frame(marker.duration, frame_rate, drop_frame),
							duration_time = luaresolve:time_from_frame(marker.duration, frame_rate).time,
							duration_frames = marker.duration,
							color = marker.color,
							note = marker.note,
							custom_data = marker.customData,
						}
                        marker_table[#marker_table+1] = data
                    end
                end
            end

            if next(marker_table) then
				-- dump(marker_table)
                export_subtitles(marker_table, "srt", file, frame_rate)
            end
            
            -- Инициализируем таблицу для этого таймлайна
            if not tlm[tln] then
                tlm[tln] = {}
            end
            
            -- по типам треков
            local trackTypes = {"video", "audio"}
            for _, trackType in ipairs(trackTypes) do
                local tc = timeline:GetTrackCount(trackType)
                
                -- Перебираем все треки этого типа
                for j = 1, tc do
                    local tlis = timeline:GetItemListInTrack(trackType, j)
                    
                    -- Перебираем все элементы в треке
                    for _, tli in ipairs(tlis) do
                        local mpi = tli:GetMediaPoolItem()
                        local mi = mpi:GetMediaId()
                        
                        -- Сохраняем клип по mediaId
                        tlm[tln][mi] = mpi
                    end
                end
            end
        end
    end

    if tlc > 1 then
        Project:SetCurrentTimeline(ctl)
    end

    
    -- Экспортируем метаданные для каждого таймлайна
    for tln, tlClip in pairs(tlClips) do
        -- local tln = tlClip:GetName()
        if all or ctlName == tln then
			local name = sanitizeFilename(tln)
			local file = output .. "/" .. name .. ".csv"
			
			-- Собираем клипы для экспорта (только из текущего таймлайна)
			local exportClips = {tlClip}
			
			if tlm[tln] then
				for mi, mpi in pairs(tlm[tln]) do
					table.insert(exportClips, mpi)
				end
			end
			
			-- Экспортируем метаданные
			print("Метаданные ~> ", file)
			if MediaPool:ExportMetadata(file, exportClips) then
				Exported = Exported+1
			end
        end
    end
    print(string.format("=== Экспортировано %d из %d ===", Exported,  tlc))
	if resolve:GetCurrentPage()~=Page then
		resolve:OpenPage(Page)
	end
end

function TrimFrame(file, ext)
    return file:gsub("(_?%d%d%d%d%d%d%d%d)(%." .. ext .. ")$", "%2")
end

-- https://github.com/rogmag/rogmag.github.io

local function write_file(filename, content)
	if Windows and HasNonAscii(filename) then
		local temp=createTempFile("")
		if temp then
		    local file_handle = temp.file
			assert(file_handle:write(content))
			assert(file_handle:close())

			rename(temp.name, filename)
			temp:close()
			return
		end
	end
	local file_handle = assert(io.open(filename, "w+"))
    assert(file_handle:write(content))
    assert(file_handle:close())
end

local function export(marker_table, config, filename)
    local export_content = {}

    if not config.hide_header then
        local header = {}

        for _, export_field in ipairs(config.fields) do
            header[#header+1] = export_field.name
        end
    
        export_content[#export_content+1] = table.concat(header, config.separator)
    end

    for _, marker_data in ipairs(marker_table) do
        local row = {}

        for _, export_field in ipairs(config.fields) do
            row[#row+1] = export_field:value(marker_data)
        end

        export_content[#export_content+1] = table.concat(row, config.separator)
    end

    write_file(filename, table.concat(export_content, "\n"))
end

local function export_json(marker_table, filename)
    local json = require ("dkjson")
    write_file(filename, json.encode(marker_table, { indent = true }))
end

local function export_youtube_chapters(marker_table, filename)
    local show_hours = luaresolve:time_from_frame(marker_table[#marker_table].frame, frame_rate).hours > 0
    local has_first_chapter = false
    local chapters = {}
        
    for index, marker_data in ipairs(marker_table) do
        local time_info = luaresolve:time_from_frame(marker_data.frame, frame_rate)

        if not has_first_chapter then
            if time_info.total_milliseconds > 0 then
                -- If there's no marker at the start of the timeline we add a chapter
                chapters[#chapters+1] = iif(show_hours, "00:00:00", "00:00").." Start"
            end

            has_first_chapter = true
        end

        if show_hours then
            chapters[#chapters+1] = string.format("%02d:%02d:%02d %s", time_info.hours, time_info.minutes, time_info.seconds, marker_data.name)
        else
            chapters[#chapters+1] = string.format("%02d:%02d %s", time_info.minutes, time_info.seconds, marker_data.name)
        end
    end

    write_file(filename, table.concat(chapters, "\n"))
end

function export_subtitles(marker_table, format, filename, frame_rate)
    local subtitles = {}

    if format == "webvtt" then
        subtitles[#subtitles+1] = "WEBVTT\n"
    end

    for index, marker_data in ipairs(marker_table) do
        local start_time = luaresolve:time_from_frame(marker_data.frame, frame_rate).time
        local end_time = luaresolve:time_from_frame(marker_data.frame + marker_data.duration_frames, frame_rate).time

        if format == "srt" then
            start_time = start_time:gsub("%.", ",")
            end_time = end_time:gsub("%.", ",")
        end

        subtitles[#subtitles+1] = string.format("%s\n%s --> %s\n%s\n", index, start_time, end_time, marker_data.note)
    end

    write_file(filename, table.concat(subtitles, "\n"))
end

stringex =
{
	quote_if_needed = function(val, force)
		local val_str = tostring(val)
		local needs_quote = val_str:find("\t") or val_str:find("\n") or val_str:find("\"")
		local escaped_val_str = val_str:gsub("\"", "\"\"")
		
		return iif(needs_quote or force, "\""..escaped_val_str.."\"", escaped_val_str)
	end,
}

luaresolve = 
{
	frame_rates =
	{
		get_fraction = function(self, frame_rate_string_or_number)
			local frame_rate = tonumber(tostring(frame_rate_string_or_number))
			-- These are the frame rates that DaVinci Resolve Studio supports as of version 18
			local frame_rates = { 16, 18, 23.976, 24, 25, 29.97, 30, 47.952, 48, 50, 59.94, 60, 72, 95.904, 96, 100, 119.88, 120 }

			for _, current_frame_rate in ipairs (frame_rates) do
				if current_frame_rate == frame_rate or math.floor(current_frame_rate) == frame_rate then
					local is_decimal = current_frame_rate % 1 > 0
					local denominator = iif(is_decimal, 1001, 100)
					local numerator = math.ceil(current_frame_rate) * iif(is_decimal, 1000, denominator)
					return { num = numerator, den = denominator }
				end
			end

			return nil, string.format("Invalid frame rate: %s", frame_rate_string_or_number)
		end,

		get_decimal = function(self, frame_rate_string_or_number)
			local fractional_frame_rate, error_message = self:get_fraction(frame_rate_string_or_number)
			
			if fractional_frame_rate ~= nil then
				return tonumber(string.format("%.3f", fractional_frame_rate.num / fractional_frame_rate.den))
			else
				return nil, error_message
			end
		end,
	},

	load_library = function(name_pattern)
		local files = bmd.readdir(fu:MapPath("FusionLibs:"..iif(ffi.os == "Windows", "", "../"))..name_pattern)
		assert(#files == 1 and files[1].IsDir == false, string.format("Couldn't find exact match for pattern \"%s.\"", name_pattern))
		return ffi.load(files.Parent..files[1].Name)
	end,

	frame_from_timecode = function(self, timecode, frame_rate)
		return libavutil:av_timecode_init_from_string(timecode, self.frame_rates:get_fraction(frame_rate)).start
	end,

	timecode_from_frame = function(self, frame, frame_rate, drop_frame)
		return libavutil:av_timecode_make_string(0, frame, self.frame_rates:get_decimal(frame_rate),
		{
			AV_TIMECODE_FLAG_DROPFRAME = drop_frame == true or drop_frame == 1 or drop_frame == "1",
			AV_TIMECODE_FLAG_24HOURSMAX = true,
			AV_TIMECODE_FLAG_ALLOWNEGATIVE = false
		})
	end,

	time_from_frame = function(self, frame, frame_rate)
		local fractional_frame_rate = self.frame_rates:get_fraction(frame_rate)

		-- Time is rounded to the nearest millisecond
		local total_milliseconds = tonumber(string.format("%.0f", 1000 * frame / (fractional_frame_rate.num / fractional_frame_rate.den)))
		
		local hours = math.floor(total_milliseconds / 1000 / 60 / 60)
		local minutes = math.floor(total_milliseconds / 1000 / 60) % 60
		local seconds = math.floor(total_milliseconds / 1000) % 60
		local milliseconds = total_milliseconds % 1000
		
		return
		{
			time = string.format("%02d:%02d:%02d.%03d", hours, minutes, seconds, milliseconds),
			hours = hours,
			minutes = minutes,
			seconds = seconds,
			milliseconds = milliseconds,
			total_milliseconds = total_milliseconds,
		}
	end,

	markers =
	{
		get_marker_count_by_color = function(markers)
			local marker_count_by_color = { Any = 0 }

			for marker_frame, marker in pairs(markers) do
				if marker_count_by_color[marker.color] == nil then
					marker_count_by_color[marker.color] = 0
				end

				marker_count_by_color[marker.color] = marker_count_by_color[marker.color] + 1
				marker_count_by_color.Any = marker_count_by_color.Any + 1
			end

			return marker_count_by_color
		end,

		colors = 
		{
			"Blue",
			"Cyan",
			"Green",
			"Yellow",
			"Red",
			"Pink",
			"Purple",
			"Fuchsia",
			"Rose",
			"Lavender",
			"Sky",
			"Mint",
			"Lemon",
			"Sand",
			"Cocoa",
			"Cream"
		},

		formats = 
		{
			tab_delimited_text =
			{	-- This format isn't emulating any existing format for marker exports. In addition to common
				-- fields it adds "Duration (Time)" which isn't normally available in Resolve.
				-- The exported file can be opened directly in Excel without any special settings or parsing. For Google Spreadsheet... //TODO
				name = "Tab-delimited Text",
				extension = "txt",
				export_config =
				{
					fields = table.pack
					(
						{ name = "#",				value_field = "index",			value = function(self, marker_data) return marker_data[self.value_field] end },
						{ name = "Frame",			value_field = "frame",			value = function(self, marker_data) return marker_data[self.value_field] end },
						{ name = "Name",			value_field = "name",			value = function(self, marker_data) return stringex.quote_if_needed(marker_data[self.value_field]) end },
						{ name = "Start TC",		value_field = "start_tc",		value = function(self, marker_data) return stringex.quote_if_needed(marker_data[self.value_field], true) end },
						{ name = "End TC",			value_field = "end_tc",			value = function(self, marker_data) return stringex.quote_if_needed(marker_data[self.value_field], true) end },
						{ name = "Duration (TC)",	value_field = "duration_tc",	value = function(self, marker_data) return stringex.quote_if_needed(marker_data[self.value_field], true) end },
						{ name = "Duration (Time)",	value_field = "duration_time",	value = function(self, marker_data) return marker_data[self.value_field] end },
						{ name = "Color",			value_field = "color",			value = function(self, marker_data) return marker_data[self.value_field] end },
						{ name = "Note",			value_field = "note",			value = function(self, marker_data) return stringex.quote_if_needed(marker_data[self.value_field]) end }
					),
					separator = "\t",
				},
			},

			premiere_csv = 
			{	-- Adobe calls it csv even though its tab-delimited
				name = "Premiere Pro CSV",
				extension = "csv",
				export_config =
				{
					fields = table.pack
					(
						{ name = "Marker Name",		value_field = "name",			value = function(self, marker_data) return stringex.quote_if_needed(marker_data[self.value_field]) end },
						{ name = "Description",		value_field = "note",			value = function(self, marker_data) return stringex.quote_if_needed(marker_data[self.value_field]) end },
						{ name = "In",				value_field = "start_tc",		value = function(self, marker_data) return marker_data[self.value_field] end },
						{ name = "Out",				value_field = "end_tc",			value = function(self, marker_data) return marker_data[self.value_field] end },
						{ name = "Duration",		value_field = "duration_tc",	value = function(self, marker_data) return marker_data[self.value_field] end },
						{ name = "Marker Type",		value_field = nil,				value = function(self, marker_data) return "Comment" end }
					),
					separator = "\t",
				},
			},

			json =
			{
				name = "JSON Data",
				extension = "json",
			},

			youtube =
			{
				name = "YouTube Chapters",
				extension = "txt",
				always_start_at_zero = true,
			},

			fairlight =
			{	-- Note: Fairlight can't import ADR Cues with timecode that has three digits for frames (frame rates above 100fps)
				name = "Fairlight ADR Cues",
				extension = "csv",
				export_config =
				{
					fields = table.pack
					(
						{ name = "Cue ID",			value_field = "index",			value = function(self, marker_data) return stringex.quote_if_needed(marker_data[self.value_field], true) end },
						{ name = "In Point",		value_field = "start_tc",		value = function(self, marker_data) return stringex.quote_if_needed(marker_data[self.value_field], true) end },
						{ name = "Out Point",		value_field = "end_tc",			value = function(self, marker_data) return stringex.quote_if_needed(marker_data[self.value_field], true) end },
						{ name = "Character",		value_field = "name",			value = function(self, marker_data) return stringex.quote_if_needed(marker_data[self.value_field], true) end },
						{ name = "Dialog",			value_field = "note",			value = function(self, marker_data) return stringex.quote_if_needed(marker_data[self.value_field]:gsub("\n", "<br>"), true) end },
						{ name = "Done",			value_field = nil,				value = function(self, marker_data) return stringex.quote_if_needed("False", true) end }
					),
					separator = ",",
					hide_header = true,
				},
			},

			srt =
			{
				name = "SRT Subtitles",
				extension = "srt",
			},

			webvtt =
			{
				name = "WebVTT Subtitles",
				extension = "vtt",
			},

			sort_order = { "fairlight", "json", "premiere_csv", "srt", "tab_delimited_text", "webvtt", "youtube" },
		},
	},
}

libavutil = 
{
	library = luaresolve.load_library(iif(ffi.os == "Windows", "avutil*.dll", iif(ffi.os == "OSX", "libavutil*.dylib", "libavutil.so"))),

	demand_version = function(self, version)
		local library_version = self:av_version_info()

		return (library_version.major > version.major)
			or (library_version.major == version.major and library_version.minor > version.minor)
			or (library_version.major == version.major and library_version.minor == version.minor and library_version.patch > version.patch)
			or (library_version.major == version.major and library_version.minor == version.minor and library_version.patch == version.patch)
	end,

	set_declarations = function()
		ffi.cdef[[
			enum AVTimecodeFlag {
				AV_TIMECODE_FLAG_DROPFRAME      = 1<<0, // timecode is drop frame
				AV_TIMECODE_FLAG_24HOURSMAX     = 1<<1, // timecode wraps after 24 hours
				AV_TIMECODE_FLAG_ALLOWNEGATIVE  = 1<<2, // negative time values are allowed
			};

			struct AVRational { int32_t num; int32_t den; };
			struct AVTimecode { int32_t start; enum AVTimecodeFlag flags; struct AVRational rate; uint32_t fps; };

			char* av_timecode_make_string(const struct AVTimecode* tc, const char* buf, int32_t framenum);
			int32_t av_timecode_init_from_string(struct AVTimecode* tc, struct AVRational rate, const char* str, void* log_ctx);

			char* av_version_info (void);
		]]
	end,

	av_timecode_make_string = function(self, start, frame, fps, flags)
		local function bor_number_flags(enum_name, flags)
			local enum_value = 0

			if (flags) then
				for key, value in pairs(flags) do
					if (value == true) then
						enum_value = bit.bor(enum_value, tonumber(ffi.new(enum_name, key)))
					end
				end
			end

			return enum_value;
		end

		local tc = ffi.new("struct AVTimecode",
		{
			start = start,
			flags = bor_number_flags("enum AVTimecodeFlag", flags),
			fps = math.ceil(luaresolve.frame_rates:get_decimal(fps))
		})

		if (flags.AV_TIMECODE_FLAG_DROPFRAME and fps > 60 and (fps % (30000 / 1001) == 0 or fps % 29.97 == 0))
			and (not self:demand_version( { major = 4, minor = 4, patch = 0 } ))
		then
			-- Adjust for drop frame above 60 fps (not necessary if BMD upgrades to libavutil-57 or later)
			frame = frame + 9 * tc.fps / 15 * (math.floor(frame / (tc.fps * 599.4))) + (math.floor((frame % (tc.fps * 599.4)) / (tc.fps * 59.94))) * tc.fps / 15
		end

		local timecodestring = ffi.string(self.library.av_timecode_make_string(tc, ffi.string(string.rep(" ", 16)), frame))
	
		if (#timecodestring > 0) then
			local frame_digits = #tostring(math.ceil(fps) - 1)

			-- Fix for libavutil where it doesn't use leading zeros for timecode at frame rates above 100
			if frame_digits > 2 then
				timecodestring = string.format("%s%0"..frame_digits.."d", timecodestring:sub(1, timecodestring:find("[:;]%d+$")), tonumber(timecodestring:match("%d+$")))
			end

			return timecodestring
		else
			return nil
		end
	end,

	av_timecode_init_from_string = function(self, timecode, frame_rate_fraction)
		local tc = ffi.new("struct AVTimecode")
		local result = self.library.av_timecode_init_from_string(tc, ffi.new("struct AVRational", frame_rate_fraction), timecode, ffi.new("void*", nil))
	
		if (result == 0) then
			return
			{
				start = tc.start,
				flags =
				{
					AV_TIMECODE_FLAG_DROPFRAME = bit.band(tc.flags, ffi.C.AV_TIMECODE_FLAG_DROPFRAME) == ffi.C.AV_TIMECODE_FLAG_DROPFRAME,
					AV_TIMECODE_FLAG_24HOURSMAX = bit.band(tc.flags, ffi.C.AV_TIMECODE_FLAG_24HOURSMAX) == ffi.C.AV_TIMECODE_FLAG_24HOURSMAX,
					AV_TIMECODE_FLAG_ALLOWNEGATIVE = bit.band(tc.flags, ffi.C.AV_TIMECODE_FLAG_ALLOWNEGATIVE) == ffi.C.AV_TIMECODE_FLAG_ALLOWNEGATIVE,
				},
				rate =
				{
					num = tc.rate.num,
					den = tc.rate.den
				},
				fps = tc.fps
			}
		else
			error("avutil error code: "..result)
		end
	end,

	av_version_info = function(self)
		local version = ffi.string(self.library.av_version_info())

		return 
		{
			major = tonumber(version:match("^%d+")),
			minor = tonumber(version:match("%.%d+"):sub(2)),
			patch = tonumber(version:match("%d+$"))
		}
	end,
}

libavutil.set_declarations()

-- Запускаем основную функцию
main()
