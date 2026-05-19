package configapply

import (
	"errors"
	"io/fs"
	"os"
	"strings"
)

// Explainer turns a raw error into user-facing "what happened / what expected /
// next steps" text. Order matters: the first matching Explainer wins, so put
// specific patterns ahead of GenericExplainer.
type Explainer interface {
	Match(err error) bool
	Explain(err error, plan *ChangePlan) (whatHappened, whatExpected string, next []NextStep)
}

// DefaultExplainers returns the built-in chain. Callers can prepend custom
// ones for domain-specific errors (e.g. Wails binding failures).
func DefaultExplainers() []Explainer {
	return []Explainer{
		DiskFullExplainer{},
		PermissionDeniedExplainer{},
		FileLockedExplainer{},
		FileNotFoundExplainer{},
		PathTooLongExplainer{},
		GenericExplainer{},
	}
}

// Explain walks the chain and returns the first match. GenericExplainer is the
// terminal so a non-nil result is always produced.
func Explain(chain []Explainer, err error, plan *ChangePlan) (string, string, []NextStep) {
	if err == nil {
		return "", "", nil
	}
	if len(chain) == 0 {
		chain = DefaultExplainers()
	}
	for _, e := range chain {
		if e.Match(err) {
			return e.Explain(err, plan)
		}
	}
	return GenericExplainer{}.Explain(err, plan)
}

type DiskFullExplainer struct{}

func (DiskFullExplainer) Match(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "no space left") ||
		strings.Contains(s, "disk full") ||
		strings.Contains(s, "enospc")
}
func (DiskFullExplainer) Explain(err error, _ *ChangePlan) (string, string, []NextStep) {
	return "磁盘空间不足,写入失败。",
		"配置文件应原子写入磁盘并完成 fsync。",
		[]NextStep{
			{Label: "打开磁盘清理", Action: "open_disk_cleanup"},
			{Label: "查看占用大户", Action: "open_storage_settings"},
			{Label: "稍后重试", Action: "retry"},
		}
}

type PermissionDeniedExplainer struct{}

func (PermissionDeniedExplainer) Match(err error) bool {
	if errors.Is(err, fs.ErrPermission) {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "permission denied") || strings.Contains(s, "access is denied")
}
func (PermissionDeniedExplainer) Explain(err error, _ *ChangePlan) (string, string, []NextStep) {
	return "目标文件或目录权限不足。",
		"Switch 应能写入用户主目录下的 CLI 工具配置目录。",
		[]NextStep{
			{Label: "以管理员重启 Switch", Action: "restart_as_admin"},
			{Label: "检查目录权限", Action: "open_target_dir", Params: map[string]string{"hint": "right-click → Properties → Security"}},
			{Label: "查看详细日志", Action: "open_log"},
		}
}

type FileLockedExplainer struct{}

func (FileLockedExplainer) Match(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "being used by another process") ||
		strings.Contains(s, "sharing violation") ||
		strings.Contains(s, "the process cannot access the file")
}
func (FileLockedExplainer) Explain(err error, _ *ChangePlan) (string, string, []NextStep) {
	return "目标文件被其他进程锁住(可能是杀毒软件或代码编辑器)。",
		"重命名 temp 文件到目标路径应能成功。",
		[]NextStep{
			{Label: "关闭代码编辑器后重试", Action: "retry"},
			{Label: "暂停杀毒实时扫描", Action: "open_av_settings"},
			{Label: "查看哪个进程占用", Action: "show_lock_holder"},
		}
}

type FileNotFoundExplainer struct{}

func (FileNotFoundExplainer) Match(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}
func (FileNotFoundExplainer) Explain(err error, _ *ChangePlan) (string, string, []NextStep) {
	return "依赖的源文件或快照不存在。",
		"读取 before 状态或恢复快照时文件应存在。",
		[]NextStep{
			{Label: "查看快照列表", Action: "open_snapshots"},
			{Label: "重新打包/初始化", Action: "reinit"},
		}
}

type PathTooLongExplainer struct{}

func (PathTooLongExplainer) Match(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "filename too long") ||
		strings.Contains(s, "path too long") ||
		strings.Contains(s, "the filename or extension is too long")
}
func (PathTooLongExplainer) Explain(err error, _ *ChangePlan) (string, string, []NextStep) {
	return "目标路径超过 OS 限制(Windows 默认 260 字符)。",
		"Switch 写入的配置路径应在 OS 限制内。",
		[]NextStep{
			{Label: "启用 Windows 长路径支持", Action: "open_long_paths_doc",
				URL: "https://learn.microsoft.com/en-us/windows/win32/fileio/maximum-file-path-limitation"},
			{Label: "把项目移到更短的路径", Action: "move_project"},
		}
}

type GenericExplainer struct{}

func (GenericExplainer) Match(error) bool { return true }
func (GenericExplainer) Explain(err error, _ *ChangePlan) (string, string, []NextStep) {
	return "Switch 在应用配置时遇到了未分类的错误: " + err.Error(),
		"配置应原子写入并验证通过。",
		[]NextStep{
			{Label: "查看详细日志", Action: "open_log"},
			{Label: "重试", Action: "retry"},
			{Label: "汇报到 GitHub", Action: "open_issue",
				URL: "https://github.com/hanmahong5-arch/lurus-switch/issues"},
		}
}
