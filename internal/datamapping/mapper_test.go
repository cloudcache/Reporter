package datamapping

import (
	"testing"

	"reporter/internal/domain"
)

func TestTransformJSONWithDictionary(t *testing.T) {
	source := domain.DataSource{
		Protocol: "http",
		Dictionaries: []domain.DictionaryMapping{
			{Name: "性别", Entries: []domain.DictionaryEntry{{Key: "M", Label: "男", Value: "男"}}},
		},
		FieldMapping: []domain.FieldMapping{
			{Source: "$.id", Target: "patient.patientNo", Required: true},
			{Source: "$.name", Target: "patient.name"},
			{Source: "$.gender", Target: "patient.gender", Dictionary: "性别"},
			{Source: "$.visit.no", Target: "visit.visitNo"},
		},
	}
	records, err := Transform(source, map[string]interface{}{"id": "P1001", "name": "赵六", "gender": "M", "visit": map[string]interface{}{"no": "V1001"}})
	if err != nil {
		t.Fatal(err)
	}
	patient := records[0].Entities["patient"]
	if patient["gender"] != "男" || patient["patientNo"] != "P1001" {
		t.Fatalf("unexpected patient mapping: %#v", patient)
	}
	if records[0].Entities["visit"]["visitNo"] != "V1001" {
		t.Fatalf("unexpected visit mapping: %#v", records[0].Entities["visit"])
	}
}

func TestTransformSOAPXML(t *testing.T) {
	source := domain.DataSource{
		Protocol: "soap",
		FieldMapping: []domain.FieldMapping{
			{Source: "Patient.PatientNo", Target: "patient.patientNo"},
			{Source: "Patient.Name", Target: "patient.name"},
			{Source: "Visit.VisitNo", Target: "visit.visitNo"},
		},
	}
	payload := `<soap:Envelope><soap:Body><Patient><PatientNo>P2001</PatientNo><Name>钱七</Name></Patient><Visit><VisitNo>V2001</VisitNo></Visit></soap:Body></soap:Envelope>`
	records, err := Transform(source, payload)
	if err != nil {
		t.Fatal(err)
	}
	if records[0].Entities["patient"]["name"] != "钱七" || records[0].Entities["visit"]["visitNo"] != "V2001" {
		t.Fatalf("unexpected XML mapping: %#v", records[0].Entities)
	}
}

func TestTransformHL7AndDICOM(t *testing.T) {
	hl7Source := domain.DataSource{
		Protocol: "hl7",
		Dictionaries: []domain.DictionaryMapping{
			{Name: "HL7 性别", Entries: []domain.DictionaryEntry{{Key: "M", Label: "男", Value: "男"}}},
		},
		FieldMapping: []domain.FieldMapping{
			{Source: "PID.3", Target: "patient.patientNo"},
			{Source: "PID.5.1", Target: "patient.name"},
			{Source: "PID.8", Target: "patient.gender", Dictionary: "HL7 性别"},
		},
	}
	hl7Payload := "MSH|^~\\&|HIS|HOSP|REPORTER|HOSP|202605141200||ADT^A01|MSG001|P|2.5.1\rPID|1||P3001||孙八||19800101|M"
	records, err := Transform(hl7Source, hl7Payload)
	if err != nil {
		t.Fatal(err)
	}
	if records[0].Entities["patient"]["name"] != "孙八" || records[0].Entities["patient"]["gender"] != "男" {
		t.Fatalf("unexpected HL7 mapping: %#v", records[0].Entities)
	}

	dicomSource := domain.DataSource{
		Protocol: "dicom",
		FieldMapping: []domain.FieldMapping{
			{Source: "0010,0020", Target: "patient.patientNo"},
			{Source: "0008,1030", Target: "record.studyDesc"},
			{Source: "0020,000D", Target: "record.studyUid"},
		},
	}
	records, err = Transform(dicomSource, map[string]interface{}{"0010,0020": "P4001", "0008,1030": "胸部 CT", "0020,000D": "1.2.3"})
	if err != nil {
		t.Fatal(err)
	}
	if records[0].Entities["record"]["studyUid"] != "1.2.3" {
		t.Fatalf("unexpected DICOM mapping: %#v", records[0].Entities)
	}
}
