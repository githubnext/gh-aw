package console

import (
	"testing"

	lipgloss "charm.land/lipgloss/v2"
)

func TestWizardTitleGradientUsesPrimerGreen(t *testing.T) {
	gradient := lipgloss.Blend1D(
		3,
		lipgloss.Color(primerGreenLight),
		lipgloss.Color(primerGreenDark),
	)

	if len(gradient) != 3 {
		t.Fatalf("gradient length = %d, want 3", len(gradient))
	}

	startR, startG, startB, startA := gradient[0].RGBA()
	wantStartR, wantStartG, wantStartB, wantStartA := lipgloss.Color(primerGreenLight).RGBA()
	if startR != wantStartR || startG != wantStartG || startB != wantStartB || startA != wantStartA {
		t.Fatalf("gradient start = (%d, %d, %d, %d), want Primer green %s", startR, startG, startB, startA, primerGreenLight)
	}

	endR, endG, endB, endA := gradient[len(gradient)-1].RGBA()
	wantEndR, wantEndG, wantEndB, wantEndA := lipgloss.Color(primerGreenDark).RGBA()
	if endR != wantEndR || endG != wantEndG || endB != wantEndB || endA != wantEndA {
		t.Fatalf("gradient end = (%d, %d, %d, %d), want Primer green %s", endR, endG, endB, endA, primerGreenDark)
	}
}
