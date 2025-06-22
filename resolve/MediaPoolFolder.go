// https://resolvedevdoc.readthedocs.io/en/latest/API_basic.html#mediapoolitem
package resolve

// MediaPoolFolder представляет папку в медиапуле DaVinci Resolve
type MediaPoolFolder struct{}

// 1. Получить список клипов в папке
func (f *MediaPoolFolder) GetClipList() []*MediaPoolItem {
	return nil // []*MPI
}

// 2. Получить имя папки
func (f *MediaPoolFolder) GetName() string {
	return "" // string
}

// 3. Получить список подпапок
func (f *MediaPoolFolder) GetSubFolderList() []*MediaPoolFolder {
	return nil // []*MPF
}
