package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// collectUnknownFields porównuje surowy JSON od AI ze strukturą docelową i
// zwraca posortowane, pozbawione duplikatów ścieżki pól, których struktura nie
// zna (np. `phases[1].repeats`). Parser takie pola po cichu ignoruje, więc bez
// tej listy literówka w nazwie pola przeszłaby niezauważona.
//
// Klucze map o dowolnych nazwach (np. `week` z Monday–Sunday) NIE są nazwami
// pól, więc nigdy nie trafiają na listę — schodzimy tylko w ich wartości.
func collectUnknownFields(raw string, target any) []string {
	var decoded any
	if err := json.Unmarshal([]byte(stripJSONFence(raw)), &decoded); err != nil {
		return nil
	}
	found := map[string]bool{}
	walkUnknownFields(decoded, reflect.TypeOf(target), "", found)
	paths := make([]string, 0, len(found))
	for path := range found {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func walkUnknownFields(value any, target reflect.Type, path string, found map[string]bool) {
	for target != nil && target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	if target == nil {
		return
	}
	switch actual := value.(type) {
	case map[string]any:
		switch target.Kind() {
		case reflect.Struct:
			fields := jsonFieldTypes(target)
			for key, item := range actual {
				fieldType, known := fields[strings.ToLower(key)]
				if !known {
					found[joinFieldPath(path, key)] = true
					continue
				}
				walkUnknownFields(item, fieldType, joinFieldPath(path, key), found)
			}
		case reflect.Map:
			for key, item := range actual {
				walkUnknownFields(item, target.Elem(), joinFieldPath(path, key), found)
			}
		}
	case []any:
		if target.Kind() == reflect.Slice || target.Kind() == reflect.Array {
			for index, item := range actual {
				walkUnknownFields(item, target.Elem(), fmt.Sprintf("%s[%d]", path, index), found)
			}
		}
	}
}

// jsonFieldTypes zwraca typy pól struktury pod kluczami JSON zapisanymi małymi
// literami — encoding/json dopasowuje nazwy bez rozróżniania wielkości liter,
// więc tak samo musi działać wykrywanie nieznanych pól.
func jsonFieldTypes(target reflect.Type) map[string]reflect.Type {
	fields := map[string]reflect.Type{}
	for index := 0; index < target.NumField(); index++ {
		field := target.Field(index)
		if field.PkgPath != "" {
			continue
		}
		tag := field.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name := field.Name
		if comma := strings.Index(tag, ","); comma >= 0 {
			tag = tag[:comma]
		}
		if tag != "" {
			name = tag
		}
		fields[strings.ToLower(name)] = field.Type
	}
	return fields
}

func joinFieldPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}
