// https://resolvedevdoc.readthedocs.io/en/latest/API_basic.html#projectmanager
package resolve

type ProjectManager struct{}

// 1. ArchiveProject - архивирует проект по указанному пути
func (pm *ProjectManager) ArchiveProject(projectName, filePath string, isArchiveSrcMedia bool, isArchiveRenderCache bool, isArchiveProxyMedia bool) bool {
	return false
}

// 2. CreateProject - создает и возвращает новый проект
func (pm *ProjectManager) CreateProject(projectName string) *Project {
	return nil
}

// 3. DeleteProject - удаляет проект в текущей папке
func (pm *ProjectManager) DeleteProject(projectName string) bool {
	return false
}

// 4. LoadProject - загружает и возвращает проект по имени
func (pm *ProjectManager) LoadProject(projectName string) *Project {
	return nil
}

// 5. GetCurrentProject - возвращает текущий загруженный проект
func (pm *ProjectManager) GetCurrentProject() *Project {
	return nil
}

// 6. SaveProject - сохраняет текущий проект
func (pm *ProjectManager) SaveProject() bool {
	return false
}

// 7. CloseProject - закрывает указанный проект без сохранения
func (pm *ProjectManager) CloseProject(project *Project) bool {
	return false
}

// 8. CreateFolder - создает папку в текущей директории
func (pm *ProjectManager) CreateFolder(folderName string) bool {
	return false
}

// 9. DeleteFolder - удаляет указанную папку
func (pm *ProjectManager) DeleteFolder(folderName string) bool {
	return false
}

// 10. GetProjectListInCurrentFolder - возвращает список проектов в текущей папке
func (pm *ProjectManager) GetProjectListInCurrentFolder() []string {
	return nil
}

// 11. GetFolderListInCurrentFolder - возвращает список папок в текущей директории
func (pm *ProjectManager) GetFolderListInCurrentFolder() []string {
	return nil
}

// 12. GotoRootFolder - открывает корневую папку базы данных
func (pm *ProjectManager) GotoRootFolder() bool {
	return false
}

// 13. GotoParentFolder - открывает родительскую папку текущей директории
func (pm *ProjectManager) GotoParentFolder() bool {
	return false
}

// 14. GetCurrentFolder - возвращает имя текущей папки
func (pm *ProjectManager) GetCurrentFolder() string {
	return ""
}

// 15. OpenFolder - открывает указанную папку
func (pm *ProjectManager) OpenFolder(folderName string) bool {
	return false
}

// 16. ImportProject - импортирует проект из файла
func (pm *ProjectManager) ImportProject(filePath string, projectName ...string) bool {
	return false
}

// 17. ExportProject - экспортирует проект в файл
func (pm *ProjectManager) ExportProject(projectName, filePath string, withStillsAndLUTs bool) bool {
	return false
}

// 18. RestoreProject - восстанавливает проект из файла
func (pm *ProjectManager) RestoreProject(filePath string, projectName ...string) bool {
	return false
}

// 19. GetCurrentDatabase - возвращает информацию о текущей базе данных
func (pm *ProjectManager) GetCurrentDatabase() map[string]interface{} {
	return nil
}

// 20. GetDatabaseList - возвращает список доступных баз данных
func (pm *ProjectManager) GetDatabaseList() []map[string]interface{} {
	return nil
}

// 21. SetCurrentDatabase - устанавливает текущую базу данных
func (pm *ProjectManager) SetCurrentDatabase(dbInfo map[string]interface{}) bool {
	return false
}
