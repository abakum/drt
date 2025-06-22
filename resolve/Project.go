// https://resolvedevdoc.readthedocs.io/en/latest/API_basic.html#project
package resolve

type Project struct{}

// 1. GetMediaPool - возвращает объект MediaPool
func (p *Project) GetMediaPool() *MediaPool {
	return nil
}

// 2. GetTimelineCount - возвращает количество таймлайнов в проекте
func (p *Project) GetTimelineCount() int {
	return 0
}

// 3. GetTimelineByIndex - возвращает таймлайн по индексу (1-based)
func (p *Project) GetTimelineByIndex(idx int) *Timeline {
	return nil
}

// 4. GetCurrentTimeline - возвращает текущий таймлайн
func (p *Project) GetCurrentTimeline() *Timeline {
	return nil
}

// 5. SetCurrentTimeline - устанавливает текущий таймлайн
func (p *Project) SetCurrentTimeline(timeline *Timeline) bool {
	return false
}

// 6. GetGallery - возвращает объект Gallery
func (p *Project) GetGallery() *Gallery {
	return nil
}

// 7. GetName - возвращает имя проекта
func (p *Project) GetName() string {
	return ""
}

// 8. SetName - устанавливает имя проекта
func (p *Project) SetName(projectName string) bool {
	return false
}

// 9. GetPresetList - возвращает список пресетов
func (p *Project) GetPresetList() []interface{} {
	return nil
}

// 10. SetPreset - устанавливает пресет для проекта
func (p *Project) SetPreset(presetName string) bool {
	return false
}

// 11. AddRenderJob - добавляет задачу рендеринга
func (p *Project) AddRenderJob() string {
	return ""
}

// 12. DeleteRenderJob - удаляет задачу рендеринга
func (p *Project) DeleteRenderJob(jobId string) bool {
	return false
}

// 13. DeleteAllRenderJobs - удаляет все задачи рендеринга
func (p *Project) DeleteAllRenderJobs() bool {
	return false
}

// 14. GetRenderJobList - возвращает список задач рендеринга
func (p *Project) GetRenderJobList() []interface{} {
	return nil
}

// 15. GetRenderPresetList - возвращает список пресетов рендеринга
func (p *Project) GetRenderPresetList() []interface{} {
	return nil
}

// 16. StartRendering - запускает рендеринг указанных задач (вариативная версия)
func (p *Project) StartRendering(jobIds ...string) bool {
	return false
}

// 17. StartRenderingSlice - запускает рендеринг указанных задач (версия со срезом)
func (p *Project) StartRenderingSlice(jobIds []string, isInteractiveMode bool) bool {
	return false
}

// 18. StartRenderingAll - запускает рендеринг всех задач
func (p *Project) StartRenderingAll(isInteractiveMode bool) bool {
	return false
}

// 19. StopRendering - останавливает текущий рендеринг
func (p *Project) StopRendering() {
	// no return value
}

// 20. IsRenderingInProgress - проверяет выполняется ли рендеринг
func (p *Project) IsRenderingInProgress() bool {
	return false
}

// 21. LoadRenderPreset - загружает пресет рендеринга
func (p *Project) LoadRenderPreset(presetName string) bool {
	return false
}

// 22. SaveAsNewRenderPreset - сохраняет новый пресет рендеринга
func (p *Project) SaveAsNewRenderPreset(presetName string) bool {
	return false
}

// 23. SetRenderSettings - устанавливает параметры рендеринга
func (p *Project) SetRenderSettings(settings map[string]interface{}) bool {
	return false
}

// 24. GetRenderJobStatus - возвращает статус задачи рендеринга
func (p *Project) GetRenderJobStatus(jobId string) map[string]interface{} {
	return nil
}

// 25. GetSetting - возвращает значение настройки проекта
func (p *Project) GetSetting(settingName string) string {
	return ""
}

// 26. SetSetting - устанавливает значение настройки проекта
func (p *Project) SetSetting(settingName, settingValue string) bool {
	return false
}

// 27. GetRenderFormats - возвращает доступные форматы рендеринга
func (p *Project) GetRenderFormats() map[string]string {
	return nil
}

// 28. GetRenderCodecs - возвращает доступные кодеки для формата
func (p *Project) GetRenderCodecs(renderFormat string) map[string]string {
	return nil
}

// 29. GetCurrentRenderFormatAndCodec - возвращает текущие формат и кодек
func (p *Project) GetCurrentRenderFormatAndCodec() map[string]string {
	return nil
}

// 30. SetCurrentRenderFormatAndCodec - устанавливает формат и кодек рендеринга
func (p *Project) SetCurrentRenderFormatAndCodec(format, codec string) bool {
	return false
}

// 31. GetCurrentRenderMode - возвращает текущий режим рендеринга
func (p *Project) GetCurrentRenderMode() int {
	return 0
}

// 32. SetCurrentRenderMode - устанавливает режим рендеринга
func (p *Project) SetCurrentRenderMode(renderMode int) bool {
	return false
}

// 33. GetRenderResolutions - возвращает доступные разрешения рендеринга
func (p *Project) GetRenderResolutions(format, codec string) []map[string]int {
	return nil
}

// 34. RefreshLUTList - обновляет список LUT
func (p *Project) RefreshLUTList() bool {
	return false
}

// Вспомогательные типы
