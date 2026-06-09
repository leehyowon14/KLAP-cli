package app

import (
	"testing"

	"github.com/kw-klap/klap-cli/internal/klas"
)

func TestNormalizeRoom(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "연 102", want: "연구관102"},
		{input: "연구 102", want: "연구관102"},
		{input: "연구관102", want: "연구관102"},
		{input: "비502", want: "비마관502"},
		{input: "비마 502", want: "비마관502"},
		{input: "새 103", want: "새빛관103"},
		{input: "새빛103", want: "새빛관103"},
		{input: "한울B101", want: "한울관B101"},
	}
	for _, tt := range tests {
		if got := NormalizeRoom(tt.input); got != tt.want {
			t.Fatalf("NormalizeRoom(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeBuilding(t *testing.T) {
	if got := NormalizeBuilding("참빛"); got != "참빛관" {
		t.Fatalf("NormalizeBuilding() = %q, want %q", got, "참빛관")
	}
}

func TestMatchRoomsFiltersBuildingAlias(t *testing.T) {
	rooms := map[string][]RoomBusyRow{
		"비마관502":  {{Room: "비마관502", Weekday: 1, Period: 1}},
		"새빛관502":  {{Room: "새빛관502", Weekday: 1, Period: 2}},
		"연구관102":  {{Room: "연구관102", Weekday: 2, Period: 3}},
		"참빛관B101": {{Room: "참빛관B101", Weekday: 3, Period: 4}},
	}
	got := matchRooms(rooms, "502", "비마")
	if len(got) != 1 || got[0] != "비마관502" {
		t.Fatalf("matchRooms() = %v, want [비마관502]", got)
	}
}

func TestAddRoomIndexTimesCanonicalizesRooms(t *testing.T) {
	index := RoomIndex{Rooms: map[string][]RoomBusyRow{}}
	item := klas.SyllabusListItem{
		KoreanName:    "컴퓨터그래픽스",
		Professor:     "김동준",
		OpenMajorCode: "I040",
		OpenGrade:     "3",
		OpenGwamokNo:  "3951",
		BunbanNo:      "01",
	}

	addRoomIndexTimes(&index, item, "U202613951I040013", []klas.SyllabusTime{
		{Weekday: "월", Periods: []int{1, 2}, Room: "연102"},
		{Weekday: "화", Periods: []int{3}, Room: "비502"},
	})

	if len(index.Rooms["연구관102"]) != 2 {
		t.Fatalf("연구관102 rows = %v, want 2 rows", index.Rooms["연구관102"])
	}
	if len(index.Rooms["비마관502"]) != 1 {
		t.Fatalf("비마관502 rows = %v, want 1 row", index.Rooms["비마관502"])
	}
}
