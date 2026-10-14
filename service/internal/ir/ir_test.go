package ir

import (
	"encoding/json"
	"testing"
)

func TestMetaMarshalMergesExtraWithoutOverwritingNamedFields(t *testing.T) {
	m := Meta{
		Series: "S", Theme: "dark", Accent: "purple", Size: "landscape",
		Extra: map[string]any{"org": "acme", "series": "should not win"},
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	json.Unmarshal(b, &got)
	if got["series"] != "S" {
		t.Fatalf("named field must win over Extra, got %v", got["series"])
	}

	if got["org"] != "acme" {
		t.Fatalf("expected Extra field to be merged in, got %v", got)
	}
}

func TestMetaMarshalWithNoExtra(t *testing.T) {
	m := Meta{Series: "S", Theme: "light", Accent: "blue", Size: "og"}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	json.Unmarshal(b, &got)
	if got["series"] != "S" || got["accent"] != "blue" {
		t.Fatalf("unexpected: %s", b)
	}

	if _, present := got["author"]; present {
		t.Fatalf("omitempty fields should be absent, got %s", b)
	}
}

func TestBlockNamesIncludesImage(t *testing.T) {
	if !BlockNames["image"] {
		t.Fatal("image should be a registered block")
	}

	if BlockNames["not-a-block"] {
		t.Fatal("unregistered name should not be present")
	}
}

func TestMetaUnmarshalSeparatesNamedFromExtra(t *testing.T) {
	var m Meta
	err := json.Unmarshal([]byte(`{"series":"S","theme":"dark","accent":"blue","size":"og","org":"acme","owner":"a"}`), &m)
	if err != nil {
		t.Fatal(err)
	}

	if m.Series != "S" || m.Theme != "dark" || m.Accent != "blue" {
		t.Fatalf("unexpected named fields: %+v", m)
	}

	if m.Extra["org"] != "acme" || m.Extra["owner"] != "a" {
		t.Fatalf("unexpected extra: %+v", m.Extra)
	}

	if _, present := m.Extra["series"]; present {
		t.Fatal("named fields must not also land in Extra")
	}
}

func TestMetaUnmarshalLogoThreeStates(t *testing.T) {
	var absent Meta
	json.Unmarshal([]byte(`{"series":"S","theme":"light","accent":"purple","size":"landscape"}`), &absent)
	if absent.LogoSet {
		t.Fatal("logo not mentioned should not set LogoSet")
	}

	var explicitNull Meta
	json.Unmarshal([]byte(`{"series":"S","theme":"light","accent":"purple","size":"landscape","logo":null}`), &explicitNull)
	if !explicitNull.LogoSet || explicitNull.Logo != nil {
		t.Fatalf("explicit null should set LogoSet with nil Logo, got LogoSet=%v Logo=%v", explicitNull.LogoSet, explicitNull.Logo)
	}

	var explicitString Meta
	json.Unmarshal([]byte(`{"series":"S","theme":"light","accent":"purple","size":"landscape","logo":"/x.png"}`), &explicitString)
	if !explicitString.LogoSet || explicitString.Logo == nil || *explicitString.Logo != "/x.png" {
		t.Fatalf("unexpected: LogoSet=%v Logo=%v", explicitString.LogoSet, explicitString.Logo)
	}
}

func TestMetaRoundTripsThroughCard(t *testing.T) {
	src := `{"meta":{"series":"S","theme":"dark","accent":"blue","size":[1200,630],"org":"acme"},"slide":{"panels":[{"nodes":[]}]}}`
	var card Card
	if err := json.Unmarshal([]byte(src), &card); err != nil {
		t.Fatal(err)
	}

	if card.Meta.Series != "S" || card.Meta.Extra["org"] != "acme" {
		t.Fatalf("unexpected: %+v", card.Meta)
	}

	b, err := json.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}

	var back Card
	json.Unmarshal(b, &back)
	if back.Meta.Series != "S" || back.Meta.Extra["org"] != "acme" {
		t.Fatalf("round-trip lost data: %s", b)
	}
}
