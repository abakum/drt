// https://resolvedevdoc.readthedocs.io/en/latest/API_basic.html#mediastorage
package resolve

type MediaStorage struct{}

// 1. Получить список смонтированных томов
func (ms *MediaStorage) GetMountedVolumeList() []string {
	return nil // []string
}

// 2. Получить список подпапок в указанной папке
func (ms *MediaStorage) GetSubFolderList(folderPath string) []string {
	return nil // []string
}

// 3. Проверить, устарела ли папка (collaboration mode)
func (ms *MediaStorage) GetIsFolderStale(folderPath string) bool {
	return false // bool
}

// 4. Получить список файлов в указанной папке
func (ms *MediaStorage) GetFileList(folderPath string) []string {
	return nil // []string
}

// 5. Показать путь в медиа-хранилище
func (ms *MediaStorage) RevealInStorage(path string) bool {
	return false // bool
}

// 6. Добавить элементы в медиа-пул (вариативная версия)
func (ms *MediaStorage) AddItemListToMediaPool(paths ...string) []*MediaPoolItem {
	return nil // []*MediaPoolItem
}

// 7. Добавить элементы в медиа-пул (версия со срезом)
func (ms *MediaStorage) AddItemListToMediaPoolSlice(paths []string) []*MediaPoolItem {
	return nil // []*MediaPoolItem
}

// 8. Добавить матовые клипы для указанного элемента
func (ms *MediaStorage) AddClipMattesToMediaPool(item *MediaPoolItem, paths []string, stereoEye string) bool {
	return false // bool
}

// 9. Добавить матовые клипы на таймлайн
func (ms *MediaStorage) AddTimelineMattesToMediaPool(paths []string) []*MediaPoolItem {
	return nil // []*MediaPoolItem
}
