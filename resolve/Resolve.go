// https://resolvedevdoc.readthedocs.io/en/latest/API_basic.html#resolve
package resolve

type Resolve struct{}

// 1. Fusion - возвращает объект Fusion для работы со скриптами
func (r *Resolve) Fusion() *Fusion {
	return nil
}

// 2. GetMediaStorage - возвращает объект для работы с медиа-хранилищем
func (r *Resolve) GetMediaStorage() *MediaStorage {
	return nil
}

// 3. GetProjectManager - возвращает менеджер проектов
func (r *Resolve) GetProjectManager() *ProjectManager {
	return nil
}

// 4. OpenPage - переключает на указанную страницу
//    Допустимые значения: "media", "cut", "edit", "fusion",
//    "color", "fairlight", "deliver"
func (r *Resolve) OpenPage(pageName string) bool {
	return false
}

// 5. GetCurrentPage - возвращает текущую открытую страницу
//    Возможные значения: "media", "cut", "edit", "fusion",
//    "color", "fairlight", "deliver", ""
func (r *Resolve) GetCurrentPage() string {
	return ""
}

// 6. GetProductName - возвращает название продукта ("DaVinci Resolve")
func (r *Resolve) GetProductName() string {
	return ""
}

// 7. GetVersion - возвращает версию в виде массива [major, minor, patch, build, suffix]
func (r *Resolve) GetVersion() [5]interface{} {
	return [5]interface{}{}
}

// 8. GetVersionString - возвращает версию в строковом формате "major.minor.patch[suffix].build"
func (r *Resolve) GetVersionString() string {
	return ""
}

// 9. LoadLayoutPreset - загружает preset с указанным именем
func (r *Resolve) LoadLayoutPreset(presetName string) bool {
	return false
}

// 10. UpdateLayoutPreset - обновляет preset текущим layout'ом
func (r *Resolve) UpdateLayoutPreset(presetName string) bool {
	return false
}

// 11. ExportLayoutPreset - экспортирует preset по указанному пути
func (r *Resolve) ExportLayoutPreset(presetName, presetFilePath string) bool {
	return false
}

// 12. DeleteLayoutPreset - удаляет preset
func (r *Resolve) DeleteLayoutPreset(presetName string) bool {
	return false
}

// 13. SaveLayoutPreset - сохраняет текущий layout как preset
func (r *Resolve) SaveLayoutPreset(presetName string) bool {
	return false
}

// 14. ImportLayoutPreset - импортирует preset из файла
//     presetName - необязательный параметр
func (r *Resolve) ImportLayoutPreset(presetFilePath string, presetName ...string) bool {
	return false
}

// 15. Quit - завершает работу приложения
func (r *Resolve) Quit() {
	// no return value
}
