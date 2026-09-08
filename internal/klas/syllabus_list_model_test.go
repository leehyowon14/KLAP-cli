package klas

import (
	"context"
	"reflect"
	"testing"
)

func TestSyllabusListNormalizesWireFields(t *testing.T) {
	client := schemaResponseClient(`[{"thisYear":"2026","hakgi":"1","openMajorCode":"I040","openGrade":"3","openGwamokNo":"3951","bunbanNo":"01","gwamokKname":"과목","memberName":"교수","codeName1":"전선","sisuNum":3,"hakjumNum":"2.5","summary":"설명","closeOpt":"N","videoUrl":"https://example.com/video"}]`)
	items, err := client.SyllabusList(context.Background(), "2026,1", "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := SyllabusListItem{ThisYear: "2026", Hakgi: "1", OpenMajorCode: "I040", OpenGrade: "3", OpenGwamokNo: "3951", BunbanNo: "01", KoreanName: "과목", Professor: "교수", CourseType: "전선", CreditHours: "3", Credits: "2.5", Summary: "설명", CloseOpt: "N", VideoURL: "https://example.com/video"}
	if len(items) != 1 || !reflect.DeepEqual(items[0], want) {
		t.Fatalf("items=%#v", items)
	}
	id, err := items[0].SubjectID()
	if err != nil || id != "U202613951I040013" || items[0].CourseCode() != "I040-3-3951-01" {
		t.Fatalf("id=%s err=%v", id, err)
	}
	for _, fixture := range []string{"null", "[]"} {
		items, err := schemaResponseClient(fixture).SyllabusList(context.Background(), "2026,1", "", "")
		if err != nil || len(items) != 0 || (items == nil) != (fixture == "null") {
			t.Fatalf("%s: %#v %v", fixture, items, err)
		}
	}
}
