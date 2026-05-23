package pseudogui

// MenuAction identifies a pseudo-GUI choice (platform-agnostic).
type MenuAction string

const (
	ActionStatus        MenuAction = "status"
	ActionPing          MenuAction = "ping"
	ActionStart         MenuAction = "start"
	ActionStop          MenuAction = "stop"
	ActionReload        MenuAction = "reload"
	ActionExportLog     MenuAction = "export-log"
	ActionUpdateCheck   MenuAction = "update-check"
	ActionUpdateInstall MenuAction = "update-install"
	ActionProfiles      MenuAction = "profiles"
	ActionImport        MenuAction = "import"
	ActionChain         MenuAction = "chain"
	ActionAdapt         MenuAction = "adapt"
	ActionServiceMode   MenuAction = "service-mode"
	ActionRouteQuick    MenuAction = "route-quick"
	ActionSettings      MenuAction = "settings"
	ActionGroups        MenuAction = "groups"
	ActionDaemon        MenuAction = "daemon"
	ActionBulkPool      MenuAction = "bulk-pool"
	ActionQuit          MenuAction = "quit"
)

// ParseChoice maps user input to action (Dahusim + muhomor extensions).
func ParseChoice(raw string) (MenuAction, bool) {
	switch stringsTrimLower(raw) {
	case "1":
		return ActionStatus, true
	case "2":
		return ActionPing, true
	case "3":
		return ActionStart, true
	case "4":
		return ActionStop, true
	case "5":
		return ActionReload, true
	case "6":
		return ActionExportLog, true
	case "7":
		return ActionUpdateCheck, true
	case "8":
		return ActionUpdateInstall, true
	case "p", "profiles", "п":
		return ActionProfiles, true
	case "i", "import":
		return ActionImport, true
	case "h", "c", "chain":
		return ActionChain, true
	case "a", "adapt":
		return ActionAdapt, true
	case "m", "mode":
		return ActionServiceMode, true
	case "r", "route":
		return ActionRouteQuick, true
	case "s", "settings":
		return ActionSettings, true
	case "g", "groups", "группы":
		return ActionGroups, true
	case "d", "daemon", "демон":
		return ActionDaemon, true
	case "b", "bulk", "пул":
		return ActionBulkPool, true
	case "q", "quit", "exit", "выход":
		return ActionQuit, true
	default:
		return "", false
	}
}

func stringsTrimLower(s string) string {
	s = trimSpace(s)
	lower := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		lower[i] = c
	}
	return string(lower)
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
