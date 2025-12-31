package env

import (
	"encoding/json"
	"os"
	"path/filepath"
	//"syscall"
)

// Registry 记录所有已知的 minibox 实例路径
type Registry struct {
	Instances []string `json:"instances"`
}

func getGlobalRegistryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// 使用 ~/.config/minibox 存放全局配置，避免和环境目录 ~/.minibox 混淆
	dir := filepath.Join(home, ".config", "minibox")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "registry.json"), nil
}

// LoadRegistry 加载注册表
func LoadRegistry() (*Registry, error) {
	path, err := getGlobalRegistryPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Registry{Instances: []string{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var r Registry
	if err := json.Unmarshal(data, &r); err != nil {
		return &Registry{Instances: []string{}}, nil // 如果格式错误，返回空
	}
	return &r, nil
}

// Save 保存注册表
func (r *Registry) Save() error {
	path, err := getGlobalRegistryPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Register 注册一个实例路径（LRU：最近使用的排最前）
// 如果路径已存在，会将其移到最前面
func Register(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	r, err := LoadRegistry()
	if err != nil {
		return err
	}

	// 先移除已存在的路径（如果有）
	newInstances := make([]string, 0, len(r.Instances)+1)
	for _, inst := range r.Instances {
		if inst != absPath {
			newInstances = append(newInstances, inst)
		}
	}

	// 添加到最前面（最近使用的）
	r.Instances = append([]string{absPath}, newInstances...)
	return r.Save()
}

// GetList 获取所有注册的实例
func GetList() []string {
	r, err := LoadRegistry()
	if err != nil {
		return nil
	}
	return r.Instances
}

// FindActive 查找当前正在运行的第一个实例路径
// 返回路径，如果没有找到则返回空字符串
func FindActive() string {
	r, err := LoadRegistry()
	if err != nil {
		return ""
	}

	// 总是检查默认路径（即使用户手动删除了 ~/.minibox）
	userHome, _ := os.UserHomeDir()
	defaultPath := filepath.Join(userHome, ".minibox")

	// 合并列表，把默认路径放在最后检查（或者最前？策略问题）
	// 这里我们优先检查最近注册的列表
	candidates := r.Instances

	// 确保 defaultPath 也在检查列表中
	found := false
	for _, p := range candidates {
		if p == defaultPath {
			found = true
			break
		}
	}
	if !found {
		// 添加到末尾作为兜底
		candidates = append(candidates, defaultPath)
	}

	for _, path := range candidates {
		// 检查锁：CheckLock 返回 nil 表示正在运行
		if err := CheckLock(path); err == nil {
			return path
		}
	}
	return ""
}

// GetActives 获取所有活跃实例 (可以有多个吗？理论上只能有一个占用端口，但文件锁是独立的)
// 这里只返回第一个找到的作为主实例
func GetActives() []string {
	active := FindActive()
	if active != "" {
		return []string{active}
	}
	return []string{}
}
