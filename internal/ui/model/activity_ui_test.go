package model

import "testing"

func TestActivityDuplicatesLabel(t *testing.T) {
	if !ActivityDuplicatesLabel("Подключение…", "Подключение…") {
		t.Fatal("exact match")
	}
	if !ActivityDuplicatesLabel("Подключение…", "Подключение к серверу…") {
		t.Fatal("prefix match")
	}
	if ActivityDuplicatesLabel("Подключено", "Проверка соединения…") {
		t.Fatal("different lines should show")
	}
	if FilterActivityForDisplay("Подключение…", "TCP 64/128") != "TCP 64/128" {
		t.Fatal("keep distinct activity")
	}
	if FilterActivityForDisplay("Подключение…", "Подключение…") != "" {
		t.Fatal("filter duplicate")
	}
}
