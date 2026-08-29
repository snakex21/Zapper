package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddPersonReusesExistingPersonByName(t *testing.T) {
	application, err := NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	first, err := application.AddPerson("Adam", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := application.AddPerson("adam", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.PersonID != second.PersonID {
		t.Fatalf("dodanie osoby o tej samej nazwie utworzyło duplikat: %q vs %q", first.PersonID, second.PersonID)
	}
	if len(application.persons.Persons) != 1 {
		t.Fatalf("oczekiwano jednej osoby, jest %d", len(application.persons.Persons))
	}
}

func TestAddPersonEmptyNameRejected(t *testing.T) {
	application, err := NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.AddPerson("   ", ""); err == nil {
		t.Fatal("pusta nazwa powinna zostać odrzucona")
	}
}

func TestUpdatePersonDeactivatesPersonWithoutProfile(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	added, err := application.AddPerson("Adam", "")
	if err != nil {
		t.Fatal(err)
	}
	person, exists := application.personByIDLocked(added.PersonID)
	if !exists {
		t.Fatalf("nie znaleziono dodanej osoby %q", added.PersonID)
	}
	person.Active = false
	snapshot, err := application.UpdatePerson(person)
	if err != nil {
		t.Fatalf("osoba bez programu powinna dać się ukryć, a UpdatePerson zwrócił błąd: %v", err)
	}
	found := false
	for _, entry := range snapshot.Persons {
		if entry.ID != added.PersonID {
			continue
		}
		found = true
		if entry.Active {
			t.Fatal("osoba bez programu powinna być nieaktywna po ukryciu")
		}
	}
	if !found {
		t.Fatalf("osoba %q zniknęła ze Snapshot.Persons zamiast zostać oznaczona jako nieaktywna", added.PersonID)
	}

	reloaded, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	revived, exists := reloaded.personByIDLocked(added.PersonID)
	if !exists {
		t.Fatalf("osoba %q zniknęła z magazynu po ponownym wczytaniu", added.PersonID)
	}
	if revived.Active {
		t.Fatal("ukryta osoba nie powinna wracać jako aktywna po ponownym wczytaniu danych")
	}
}

func TestUpdatePersonKeepsPersonWithProfileVisible(t *testing.T) {
	application, err := NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	added, err := application.AddPerson("Ewa", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.CreateProfileForPerson(added.PersonID); err != nil {
		t.Fatal(err)
	}
	person, exists := application.personByIDLocked(added.PersonID)
	if !exists {
		t.Fatalf("nie znaleziono osoby %q", added.PersonID)
	}
	person.Active = false
	if _, err := application.UpdatePerson(person); err == nil {
		t.Fatal("osoba z aktywnym programem nie powinna dać się ukryć")
	}
}

// Scenariusz zgłoszony przez użytkownika: osoba dodana przez dialog "Dodaj osoby",
// bez profilu i bez zapisu konfiguracji, musi dać się ukryć od razu po dodaniu.
func TestUpdatePersonHidesFreshlyAddedPersonWithoutSavingConfig(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	added, err := application.AddPerson("test", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range application.config.Profiles {
		if profilePersonID(profile) == added.PersonID {
			t.Fatalf("świeżo dodana osoba %q dostała profil-widmo w a.config", added.PersonID)
		}
	}
	person, exists := application.personByIDLocked(added.PersonID)
	if !exists {
		t.Fatalf("nie znaleziono świeżo dodanej osoby %q", added.PersonID)
	}
	person.Active = false
	if _, err := application.UpdatePerson(person); err != nil {
		t.Fatalf("świeżo dodana osoba bez programu powinna dać się usunąć: %v", err)
	}
	reloaded, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	revived, exists := reloaded.personByIDLocked(added.PersonID)
	if !exists {
		t.Fatalf("osoba %q zniknęła z magazynu", added.PersonID)
	}
	if revived.Active {
		t.Fatal("ukryta osoba wróciła jako aktywna po ponownym wczytaniu")
	}
}

func TestAddPersonAcceptsExplicitID(t *testing.T) {
	application, err := NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	added, err := application.AddPerson("Ola", "Ola_2026")
	if err != nil {
		t.Fatalf("jawne ID powinno zostać przyjęte: %v", err)
	}
	if added.PersonID != "ola_2026" {
		t.Fatalf("oczekiwano znormalizowanego ID %q, jest %q", "ola_2026", added.PersonID)
	}
	person, exists := application.personByIDLocked("ola_2026")
	if !exists || person.Name != "Ola" || !person.Active {
		t.Fatalf("osoba z jawnym ID nie została zapisana poprawnie: %+v (istnieje=%v)", person, exists)
	}
}

func TestAddPersonRejectsInvalidExplicitID(t *testing.T) {
	application, err := NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, badID := range []string{"ola 2026", "ola@2026", "ola.2026", "ola/2026", "ąęś", "0123456789012345678901234567890123456789X"} {
		if _, err := application.AddPerson("Ola", badID); err == nil {
			t.Fatalf("nieprawidłowe ID %q powinno zostać odrzucone", badID)
		}
	}
	if len(application.persons.Persons) != 0 {
		t.Fatalf("odrzucone ID nie powinno tworzyć osoby, jest ich %d", len(application.persons.Persons))
	}
}

func TestAddPersonRejectsExplicitIDCollision(t *testing.T) {
	application, err := NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.AddPerson("Ola", "wspolne"); err != nil {
		t.Fatal(err)
	}
	_, err = application.AddPerson("Ala", "wspolne")
	if err == nil {
		t.Fatal("kolizja ID powinna zostać odrzucona")
	}
	if !strings.Contains(err.Error(), "wspolne") {
		t.Fatalf("komunikat błędu powinien wskazywać zajęte ID, jest: %v", err)
	}
	if len(application.persons.Persons) != 1 {
		t.Fatalf("kolizja nie powinna tworzyć drugiej osoby, jest ich %d", len(application.persons.Persons))
	}
}

func TestAddPersonGeneratesIDWhenNotGiven(t *testing.T) {
	application, err := NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	added, err := application.AddPerson("Ola", "   ")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(added.PersonID, "ola_") || added.PersonID == "ola_" {
		t.Fatalf("bez jawnego ID oczekiwano wygenerowanego %q z sufiksem, jest %q", "ola_...", added.PersonID)
	}
}

func TestAddPersonExplicitIDReactivatesSamePerson(t *testing.T) {
	application, err := NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.AddPerson("Ola", "ola_2026"); err != nil {
		t.Fatal(err)
	}
	person, _ := application.personByIDLocked("ola_2026")
	person.Active = false
	if _, err := application.UpdatePerson(person); err != nil {
		t.Fatal(err)
	}
	again, err := application.AddPerson("Ola", "ola_2026")
	if err != nil {
		t.Fatalf("ponowne dodanie tej samej osoby z tym samym ID powinno ją reaktywować: %v", err)
	}
	if again.PersonID != "ola_2026" {
		t.Fatalf("oczekiwano %q, jest %q", "ola_2026", again.PersonID)
	}
	revived, _ := application.personByIDLocked("ola_2026")
	if !revived.Active {
		t.Fatal("osoba powinna zostać reaktywowana")
	}
	if len(application.persons.Persons) != 1 {
		t.Fatalf("reaktywacja nie powinna tworzyć duplikatu, osób: %d", len(application.persons.Persons))
	}
}

func TestAddPersonExplicitIDConflictsWithExistingName(t *testing.T) {
	application, err := NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	first, err := application.AddPerson("Ola", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.AddPerson("Ola", "inne_id"); err == nil {
		t.Fatalf("nazwa zajęta przez %q powinna dać czytelny błąd zamiast duplikatu", first.PersonID)
	}
}

func TestDeletePersonRemovesEntryAndArchive(t *testing.T) {
	directory := t.TempDir()
	application, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	added, err := application.AddPerson("Adam", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.CreateProfileForPerson(added.PersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := application.FinishProfile(added.PersonID); err != nil {
		t.Fatal(err)
	}
	if len(application.archive.Profiles) != 1 {
		t.Fatalf("oczekiwano 1 zarchiwizowanego programu, jest %d", len(application.archive.Profiles))
	}
	snapshot, err := application.DeletePerson(added.PersonID)
	if err != nil {
		t.Fatal(err)
	}
	for _, person := range snapshot.Persons {
		if person.ID == added.PersonID {
			t.Fatal("usunięta osoba nadal jest w Snapshot.Persons")
		}
	}
	for _, archived := range snapshot.Archive {
		t.Fatalf("archiwum usuniętej osoby zostało w Snapshot.Archive: %s", archived.ArchiveID)
	}
	reloaded, err := NewApplication(directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := reloaded.personByIDLocked(added.PersonID); exists {
		t.Fatal("usunięta osoba wróciła po ponownym wczytaniu danych")
	}
	if len(reloaded.archive.Profiles) != 0 {
		t.Fatalf("archiwum usuniętej osoby wróciło po ponownym wczytaniu: %d wpisów", len(reloaded.archive.Profiles))
	}
	entries, err := os.ReadDir(application.archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("foldery archiwum usuniętej osoby zostały na dysku: %d", len(entries))
	}
	var store PersonStore
	if err := loadJSON(application.personsPath, &store); err != nil {
		t.Fatal(err)
	}
	for _, person := range store.Persons {
		if person.ID == added.PersonID {
			t.Fatal("usunięta osoba została w persons.json")
		}
	}
}

func TestDeletePersonRejectedWithActiveProfile(t *testing.T) {
	application, err := NewApplication(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	added, err := application.AddPerson("Ewa", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.CreateProfileForPerson(added.PersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := application.DeletePerson(added.PersonID); err == nil {
		t.Fatal("osoba z aktywnym programem nie powinna dać się usunąć")
	}
	if _, exists := application.personByIDLocked(added.PersonID); !exists {
		t.Fatal("odrzucone usunięcie nie powinno kasować osoby")
	}
}

// Regresja UI: potwierdzenie usunięcia osoby nie może wygasać samo,
// bo wtedy wolniejsze drugie kliknięcie nigdy nie usuwa osoby.
func TestPersonDeleteConfirmationDoesNotExpire(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("web", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(source)
	start := strings.Index(body, "async function deleteEditedPerson(")
	if start < 0 {
		t.Fatal("nie znaleziono deleteEditedPerson w app.js")
	}
	end := strings.Index(body[start:], "\nfunction ")
	if end < 0 {
		end = len(body) - start
	}
	function := body[start : start+end]
	if strings.Contains(function, "armDestructive(") {
		t.Fatal("deleteEditedPerson nadal używa armDestructive z wygasającym oknem 4 s — wolniejsze drugie kliknięcie nie usunie osoby")
	}
	if !strings.Contains(function, "pendingPersonDeletion") {
		t.Fatal("deleteEditedPerson powinien używać trwałego potwierdzenia pendingPersonDeletion")
	}
}
