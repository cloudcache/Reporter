package datamapping

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"reporter/internal/domain"
)

func Transform(source domain.DataSource, payload interface{}) ([]domain.MappedRecord, error) {
	docs, err := normalizePayload(source, payload)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return []domain.MappedRecord{}, nil
	}
	records := make([]domain.MappedRecord, 0, len(docs))
	for _, doc := range docs {
		entities := map[string]map[string]interface{}{}
		for _, mapping := range source.FieldMapping {
			entity, field := targetParts(mapping)
			if entity == "" || field == "" {
				continue
			}
			value, ok := extractValue(source.Protocol, doc, mapping.Source)
			if !ok || isEmpty(value) {
				value = mapping.Default
			}
			if mapping.Dictionary != "" {
				value = applyDictionary(source.Dictionaries, mapping.Dictionary, value)
			}
			if mapping.Type != "" {
				value = coerce(value, mapping.Type)
			}
			if isEmpty(value) && mapping.Required {
				return nil, fmt.Errorf("required mapping %s -> %s is empty", mapping.Source, mapping.Target)
			}
			if entities[entity] == nil {
				entities[entity] = map[string]interface{}{}
			}
			entities[entity][field] = value
		}
		records = append(records, domain.MappedRecord{Raw: doc, Entities: entities})
	}
	return records, nil
}

func Preview(source domain.DataSource, payload interface{}) (map[string]interface{}, error) {
	records, err := Transform(source, payload)
	if err != nil {
		return nil, err
	}
	columns := []string{}
	seen := map[string]bool{}
	for _, mapping := range source.FieldMapping {
		entity, field := targetParts(mapping)
		if entity == "" || field == "" {
			continue
		}
		column := entity + "." + field
		if !seen[column] {
			seen[column] = true
			columns = append(columns, column)
		}
	}
	rows := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		row := map[string]interface{}{}
		for entity, fields := range record.Entities {
			for field, value := range fields {
				row[entity+"."+field] = value
			}
		}
		rows = append(rows, row)
	}
	return map[string]interface{}{"columns": columns, "rows": rows, "records": records}, nil
}

func ApplyPatientFields(fields map[string]interface{}, patient domain.Patient) domain.Patient {
	if value := stringField(fields, "id"); value != "" {
		patient.ID = value
	}
	if value := stringField(fields, "patientNo"); value != "" {
		patient.PatientNo = value
	}
	if value := stringField(fields, "medicalRecordNo"); value != "" {
		patient.MedicalRecordNo = value
	}
	if value := stringField(fields, "name"); value != "" {
		patient.Name = value
	}
	if value := stringField(fields, "gender"); value != "" {
		patient.Gender = value
	}
	if value := stringField(fields, "birthDate"); value != "" {
		patient.BirthDate = value
	}
	if value := intField(fields, "age"); value != 0 {
		patient.Age = value
	}
	if value := stringField(fields, "idCardNo"); value != "" {
		patient.IDCardNo = value
	}
	if value := stringField(fields, "phone"); value != "" {
		patient.Phone = value
	}
	if value := stringField(fields, "address"); value != "" {
		patient.Address = value
	}
	if value := stringField(fields, "nationality"); value != "" {
		patient.Nationality = value
	}
	if value := stringField(fields, "ethnicity"); value != "" {
		patient.Ethnicity = value
	}
	if value := stringField(fields, "maritalStatus"); value != "" {
		patient.MaritalStatus = value
	}
	if value := stringField(fields, "insuranceType"); value != "" {
		patient.InsuranceType = value
	}
	if value := stringField(fields, "bloodType"); value != "" {
		patient.BloodType = value
	}
	if value := stringSliceField(fields, "allergies"); len(value) > 0 {
		patient.Allergies = value
	}
	if value := stringField(fields, "emergencyContact"); value != "" {
		patient.EmergencyContact = value
	}
	if value := stringField(fields, "emergencyPhone"); value != "" {
		patient.EmergencyPhone = value
	}
	if value := stringField(fields, "diagnosis"); value != "" {
		patient.Diagnosis = value
	}
	if value := stringField(fields, "status"); value != "" {
		patient.Status = value
	}
	if value := stringField(fields, "lastVisitAt"); value != "" {
		patient.LastVisitAt = value
	}
	if patient.Status == "" {
		patient.Status = "active"
	}
	return patient
}

func ApplyVisitFields(fields map[string]interface{}, visit domain.ClinicalVisit) domain.ClinicalVisit {
	visit.PatientID = firstNonEmpty(stringField(fields, "patientId"), visit.PatientID)
	visit.VisitNo = firstNonEmpty(stringField(fields, "visitNo"), visit.VisitNo)
	visit.VisitType = firstNonEmpty(stringField(fields, "visitType"), visit.VisitType)
	visit.DepartmentCode = firstNonEmpty(stringField(fields, "departmentCode"), visit.DepartmentCode)
	visit.DepartmentName = firstNonEmpty(stringField(fields, "departmentName"), visit.DepartmentName)
	visit.Ward = firstNonEmpty(stringField(fields, "ward"), visit.Ward)
	visit.BedNo = firstNonEmpty(stringField(fields, "bedNo"), visit.BedNo)
	visit.AttendingDoctor = firstNonEmpty(stringField(fields, "attendingDoctor"), visit.AttendingDoctor)
	visit.VisitAt = firstNonEmpty(stringField(fields, "visitAt"), visit.VisitAt)
	visit.DischargeAt = firstNonEmpty(stringField(fields, "dischargeAt"), visit.DischargeAt)
	visit.DiagnosisCode = firstNonEmpty(stringField(fields, "diagnosisCode"), visit.DiagnosisCode)
	visit.DiagnosisName = firstNonEmpty(stringField(fields, "diagnosisName"), visit.DiagnosisName)
	visit.Status = firstNonEmpty(stringField(fields, "status"), visit.Status)
	if visit.Status == "" {
		visit.Status = "active"
	}
	return visit
}

func ApplyMedicalRecordFields(fields map[string]interface{}, record domain.MedicalRecord) domain.MedicalRecord {
	record.PatientID = firstNonEmpty(stringField(fields, "patientId"), record.PatientID)
	record.VisitID = firstNonEmpty(stringField(fields, "visitId"), record.VisitID)
	record.RecordNo = firstNonEmpty(stringField(fields, "recordNo"), record.RecordNo)
	record.RecordType = firstNonEmpty(stringField(fields, "recordType"), record.RecordType)
	record.Title = firstNonEmpty(stringField(fields, "title"), record.Title)
	record.Summary = firstNonEmpty(stringField(fields, "summary"), record.Summary)
	record.ChiefComplaint = firstNonEmpty(stringField(fields, "chiefComplaint"), record.ChiefComplaint)
	record.PresentIllness = firstNonEmpty(stringField(fields, "presentIllness"), record.PresentIllness)
	record.DiagnosisCode = firstNonEmpty(stringField(fields, "diagnosisCode"), record.DiagnosisCode)
	record.DiagnosisName = firstNonEmpty(stringField(fields, "diagnosisName"), record.DiagnosisName)
	record.ProcedureName = firstNonEmpty(stringField(fields, "procedureName"), record.ProcedureName)
	record.StudyUID = firstNonEmpty(stringField(fields, "studyUid"), record.StudyUID)
	record.StudyDesc = firstNonEmpty(stringField(fields, "studyDesc"), record.StudyDesc)
	record.RecordedAt = firstNonEmpty(stringField(fields, "recordedAt"), record.RecordedAt)
	return record
}

func normalizePayload(source domain.DataSource, payload interface{}) ([]map[string]interface{}, error) {
	if payload == nil {
		payload = samplePayload(source)
	}
	switch typed := payload.(type) {
	case string:
		return parseStringPayload(source.Protocol, typed)
	case map[string]interface{}:
		return rowsFromDocument(source, typed), nil
	case []interface{}:
		rows := make([]map[string]interface{}, 0, len(typed))
		for _, item := range typed {
			if row, ok := item.(map[string]interface{}); ok {
				rows = append(rows, row)
			}
		}
		return rows, nil
	default:
		content, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		var doc interface{}
		if err := json.Unmarshal(content, &doc); err != nil {
			return nil, err
		}
		return normalizePayload(source, doc)
	}
}

func parseStringPayload(protocol, content string) ([]map[string]interface{}, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return []map[string]interface{}{}, nil
	}
	if strings.HasPrefix(content, "{") || strings.HasPrefix(content, "[") {
		var decoded interface{}
		if err := json.Unmarshal([]byte(content), &decoded); err != nil {
			return nil, err
		}
		return normalizePayload(domain.DataSource{Protocol: protocol}, decoded)
	}
	switch strings.ToLower(protocol) {
	case "soap", "xml", "http":
		doc, err := parseXML(content)
		if err != nil {
			return nil, err
		}
		return []map[string]interface{}{doc}, nil
	case "hl7":
		return []map[string]interface{}{parseHL7(content)}, nil
	case "dicom":
		return []map[string]interface{}{parseDICOMText(content)}, nil
	default:
		return []map[string]interface{}{{"value": content}}, nil
	}
}

func rowsFromDocument(source domain.DataSource, doc map[string]interface{}) []map[string]interface{} {
	rowPath := configString(source.Config, "rowPath")
	if rowPath != "" {
		if value, ok := extractJSON(doc, rowPath); ok {
			if rows, ok := value.([]interface{}); ok {
				result := make([]map[string]interface{}, 0, len(rows))
				for _, row := range rows {
					if mapped, ok := row.(map[string]interface{}); ok {
						result = append(result, mapped)
					}
				}
				return result
			}
		}
	}
	return []map[string]interface{}{doc}
}

func extractValue(protocol string, doc map[string]interface{}, path string) (interface{}, bool) {
	if path == "" {
		return nil, false
	}
	switch strings.ToLower(protocol) {
	case "hl7":
		return extractHL7(doc, path)
	case "dicom":
		return extractDICOM(doc, path)
	case "soap", "xml":
		return extractXML(doc, path)
	default:
		if value, ok := extractJSON(doc, path); ok {
			return value, true
		}
		return extractXML(doc, path)
	}
}

func extractJSON(doc map[string]interface{}, path string) (interface{}, bool) {
	path = strings.TrimPrefix(strings.TrimSpace(path), "$.")
	path = strings.TrimPrefix(path, "$")
	path = strings.ReplaceAll(path, "[]", "")
	path = strings.Trim(path, ".")
	if path == "" {
		return doc, true
	}
	var current interface{} = doc
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSpace(part)
		switch typed := current.(type) {
		case map[string]interface{}:
			next, ok := typed[part]
			if !ok {
				return nil, false
			}
			current = next
			if slice, ok := current.([]interface{}); ok && len(slice) > 0 {
				current = slice[0]
			}
		case []interface{}:
			if len(typed) == 0 {
				return nil, false
			}
			current = typed[0]
		default:
			return nil, false
		}
	}
	return current, true
}

func parseXML(content string) (map[string]interface{}, error) {
	decoder := xml.NewDecoder(strings.NewReader(content))
	stack := []string{}
	result := map[string]interface{}{}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
		switch typed := token.(type) {
		case xml.StartElement:
			stack = append(stack, stripNS(typed.Name.Local))
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			value := strings.TrimSpace(string(typed))
			if value == "" {
				continue
			}
			result[strings.Join(stack, ".")] = value
			if len(stack) > 0 {
				result[stack[len(stack)-1]] = value
			}
		}
	}
	return result, nil
}

func extractXML(doc map[string]interface{}, path string) (interface{}, bool) {
	path = strings.TrimPrefix(path, "//")
	path = strings.ReplaceAll(path, "/", ".")
	path = stripPathPrefixes(path)
	if value, ok := doc[path]; ok {
		return value, true
	}
	suffix := "." + path
	for key, value := range doc {
		if strings.HasSuffix(stripPathPrefixes(key), suffix) || stripPathPrefixes(key) == path {
			return value, true
		}
	}
	return nil, false
}

func parseHL7(content string) map[string]interface{} {
	doc := map[string]interface{}{}
	lines := strings.FieldsFunc(content, func(r rune) bool { return r == '\n' || r == '\r' })
	for _, line := range lines {
		fields := strings.Split(line, "|")
		if len(fields) == 0 {
			continue
		}
		segment := strings.TrimSpace(fields[0])
		if segment == "" {
			continue
		}
		doc[segment] = fields
		for index := 1; index < len(fields); index++ {
			doc[fmt.Sprintf("%s.%d", segment, index)] = fields[index]
			for componentIndex, component := range strings.Split(fields[index], "^") {
				doc[fmt.Sprintf("%s.%d.%d", segment, index, componentIndex+1)] = component
			}
		}
	}
	return doc
}

func extractHL7(doc map[string]interface{}, path string) (interface{}, bool) {
	if value, ok := doc[path]; ok {
		return value, true
	}
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return nil, false
	}
	segment, ok := doc[parts[0]].([]string)
	if !ok {
		return nil, false
	}
	fieldIndex, err := strconv.Atoi(parts[1])
	if err != nil || fieldIndex >= len(segment) {
		return nil, false
	}
	value := segment[fieldIndex]
	if len(parts) > 2 {
		componentIndex, err := strconv.Atoi(parts[2])
		components := strings.Split(value, "^")
		if err == nil && componentIndex > 0 && componentIndex <= len(components) {
			value = components[componentIndex-1]
		}
	}
	return value, true
}

func parseDICOMText(content string) map[string]interface{} {
	doc := map[string]interface{}{}
	for _, line := range strings.Split(content, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			key, value, ok = strings.Cut(line, ":")
		}
		if ok {
			doc[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return doc
}

func extractDICOM(doc map[string]interface{}, path string) (interface{}, bool) {
	if value, ok := doc[path]; ok {
		return value, true
	}
	normalized := normalizeDICOMTag(path)
	for key, value := range doc {
		if normalizeDICOMTag(key) == normalized {
			return value, true
		}
	}
	return extractJSON(doc, path)
}

func applyDictionary(dictionaries []domain.DictionaryMapping, name string, value interface{}) interface{} {
	raw := fmt.Sprint(value)
	for _, dictionary := range dictionaries {
		if dictionary.Name != name {
			continue
		}
		for _, entry := range dictionary.Entries {
			if entry.Key == raw || entry.Value == raw || entry.Label == raw {
				if entry.Value != "" {
					return entry.Value
				}
				return entry.Label
			}
		}
	}
	return value
}

func targetParts(mapping domain.FieldMapping) (string, string) {
	target := strings.TrimSpace(mapping.Target)
	entity := strings.TrimSpace(mapping.Entity)
	if strings.Contains(target, ".") {
		parts := strings.SplitN(target, ".", 2)
		if entity == "" {
			entity = parts[0]
		}
		target = parts[1]
	}
	if entity == "" {
		entity = "patient"
	}
	return entity, target
}

func samplePayload(source domain.DataSource) interface{} {
	switch strings.ToLower(source.Protocol) {
	case "hl7":
		return "MSH|^~\\&|HIS|HOSP|REPORTER|HOSP|202605141200||ADT^A01|MSG001|P|2.5.1\rPID|1||P9001||赵六||19800101|M|||北京市朝阳区||13900009999\rPV1|1|O|CARD^心内科^1||||1001^王医生|||||||||||V20260514001|||||||||||||||||||||||||202605141130"
	case "dicom":
		return map[string]interface{}{"0010,0020": "P9001", "0010,0010": "赵六", "0008,0050": "ACC001", "0008,1030": "胸部 CT", "0020,000D": "1.2.840.113619.2"}
	case "soap", "xml":
		return "<Envelope><Body><Patient><PatientNo>P9001</PatientNo><Name>赵六</Name><Gender>M</Gender><Phone>13900009999</Phone></Patient><Visit><VisitNo>V20260514001</VisitNo><DepartmentName>心内科</DepartmentName></Visit></Body></Envelope>"
	default:
		return map[string]interface{}{
			"id": "P9001", "name": "赵六", "gender": "M", "phone": "13900009999", "age": 46,
			"visit": map[string]interface{}{"visitNo": "V20260514001", "departmentName": "心内科", "diagnosisName": "高血压"},
		}
	}
}

func stringField(fields map[string]interface{}, key string) string {
	value, ok := fields[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func intField(fields map[string]interface{}, key string) int {
	value, ok := fields[key]
	if !ok || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
		return parsed
	default:
		parsed, _ := strconv.Atoi(fmt.Sprint(typed))
		return parsed
	}
}

func stringSliceField(fields map[string]interface{}, key string) []string {
	value, ok := fields[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return typed
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				result = append(result, text)
			}
		}
		return result
	case string:
		parts := strings.FieldsFunc(typed, func(r rune) bool { return r == ',' || r == '，' || r == ';' || r == '；' })
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if text := strings.TrimSpace(part); text != "" {
				result = append(result, text)
			}
		}
		return result
	default:
		return []string{fmt.Sprint(typed)}
	}
}

func coerce(value interface{}, valueType string) interface{} {
	switch strings.ToLower(valueType) {
	case "int", "number":
		parsed, _ := strconv.Atoi(fmt.Sprint(value))
		return parsed
	case "string":
		return fmt.Sprint(value)
	case "array", "strings":
		return stringSliceField(map[string]interface{}{"value": value}, "value")
	default:
		return value
	}
}

func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) == ""
	}
	return false
}

func configString(config map[string]interface{}, key string) string {
	if config == nil {
		return ""
	}
	if value, ok := config[key]; ok && value != nil {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return ""
}

func stripNS(value string) string {
	if _, local, ok := strings.Cut(value, ":"); ok {
		return local
	}
	return value
}

func stripPathPrefixes(path string) string {
	parts := strings.Split(path, ".")
	for index, part := range parts {
		parts[index] = stripNS(part)
	}
	return strings.Join(parts, ".")
}

func normalizeDICOMTag(tag string) string {
	return strings.NewReplacer(",", "", "(", "", ")", "", " ", "").Replace(strings.ToUpper(tag))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
