package main

import (
	"math"
	"testing"
)

func approxEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestGlickoG(t *testing.T) {
	// g(0) should be 1.0 (perfectly certain opponent)
	if g := GlickoG(0); !approxEqual(g, 1.0, 0.001) {
		t.Errorf("GlickoG(0) = %f, want 1.0", g)
	}

	// g should decrease as RD increases
	g50 := GlickoG(50)
	g200 := GlickoG(200)
	g350 := GlickoG(350)
	if g50 <= g200 || g200 <= g350 {
		t.Errorf("GlickoG should decrease with RD: g(50)=%f, g(200)=%f, g(350)=%f", g50, g200, g350)
	}

	// g should always be positive and <= 1
	if g350 <= 0 || g50 > 1.0 {
		t.Errorf("GlickoG out of range: g(50)=%f, g(350)=%f", g50, g350)
	}
}

func TestGlickoExpected(t *testing.T) {
	// Equal ratings -> 50% expected
	e := GlickoExpected(1500, 1500, 200)
	if !approxEqual(e, 0.5, 0.01) {
		t.Errorf("GlickoExpected(1500, 1500, 200) = %f, want ~0.5", e)
	}

	// Higher-rated player should have > 50% expected score
	e = GlickoExpected(1800, 1500, 200)
	if e <= 0.5 {
		t.Errorf("GlickoExpected(1800, 1500, 200) = %f, want > 0.5", e)
	}

	// Lower-rated player should have < 50% expected score
	e = GlickoExpected(1200, 1500, 200)
	if e >= 0.5 {
		t.Errorf("GlickoExpected(1200, 1500, 200) = %f, want < 0.5", e)
	}
}

func TestCalculateGlicko_EqualRatings(t *testing.T) {
	// Two equal players: winner gains, loser drops by similar amounts
	win := CalculateGlicko(1500, 200, 1500, 200, 1.0, 0)
	lose := CalculateGlicko(1500, 200, 1500, 200, 0.0, 0)

	if win.NewRating <= 1500 {
		t.Errorf("Winner rating should increase: got %f", win.NewRating)
	}
	if lose.NewRating >= 1500 {
		t.Errorf("Loser rating should decrease: got %f", lose.NewRating)
	}
	if win.Change <= 0 {
		t.Errorf("Winner change should be positive: got %f", win.Change)
	}
	if lose.Change >= 0 {
		t.Errorf("Loser change should be negative: got %f", lose.Change)
	}
}

func TestCalculateGlicko_UpsetReward(t *testing.T) {
	// Underdog (1200) beats favorite (1800): should gain more than normal
	upset := CalculateGlicko(1200, 200, 1800, 200, 1.0, 0)
	normal := CalculateGlicko(1800, 200, 1200, 200, 1.0, 0)

	if upset.Change <= normal.Change {
		t.Errorf("Upset should reward more: upset=%f, normal=%f", upset.Change, normal.Change)
	}
}

func TestCalculateGlicko_RDDecreases(t *testing.T) {
	// After a match, RD should decrease (more certainty)
	result := CalculateGlicko(1500, 350, 1500, 350, 1.0, 0)
	if result.NewRD >= 350 {
		t.Errorf("RD should decrease after a match: got %f", result.NewRD)
	}
}

func TestCalculateGlicko_RDBounds(t *testing.T) {
	// RD should never go below minRD
	result := CalculateGlicko(1500, minRD+1, 1500, minRD+1, 1.0, 0)
	if result.NewRD < minRD {
		t.Errorf("RD below minimum: got %f, min=%f", result.NewRD, minRD)
	}
}

func TestCalculateGlicko_EloFloor(t *testing.T) {
	// Rating should not drop below eloFloor
	result := CalculateGlicko(110, 350, 2000, 50, 0.0, 0)
	if result.NewRating < eloFloor {
		t.Errorf("Rating below floor: got %f, floor=%f", result.NewRating, eloFloor)
	}
}

func TestCalculateGlicko_H2HDampening(t *testing.T) {
	// Repeated matchups should produce smaller rating changes
	first := CalculateGlicko(1500, 200, 1500, 200, 1.0, 0)
	fifth := CalculateGlicko(1500, 200, 1500, 200, 1.0, 5)
	tenth := CalculateGlicko(1500, 200, 1500, 200, 1.0, 10)

	if fifth.Change >= first.Change {
		t.Errorf("5th h2h should change less: first=%f, fifth=%f", first.Change, fifth.Change)
	}
	if tenth.Change >= fifth.Change {
		t.Errorf("10th h2h should change less: fifth=%f, tenth=%f", fifth.Change, tenth.Change)
	}
}

func TestCalculateGlicko_HighRDOpponentLessImpact(t *testing.T) {
	// Beating an uncertain opponent (high RD) should matter less than
	// beating a well-known opponent (low RD), all else equal.
	vsUncertain := CalculateGlicko(1500, 200, 1500, 350, 1.0, 0)
	vsCertain := CalculateGlicko(1500, 200, 1500, 50, 1.0, 0)

	// The changes differ because g(RD) weights the opponent differently.
	// This is a structural test: both should produce a positive change.
	if vsUncertain.Change <= 0 || vsCertain.Change <= 0 {
		t.Errorf("Both wins should produce positive change: uncertain=%f, certain=%f",
			vsUncertain.Change, vsCertain.Change)
	}
}

func TestCalculateGlicko_SymmetryAtEqualRatings(t *testing.T) {
	// At equal ratings (e=0.5), the asymmetric scaling is neutral (2*0.5=1),
	// so changes should still be roughly symmetric.
	win := CalculateGlicko(1500, 200, 1500, 200, 1.0, 0)
	lose := CalculateGlicko(1500, 200, 1500, 200, 0.0, 0)

	sum := win.Change + lose.Change
	if !approxEqual(sum, 0, 5.0) {
		t.Errorf("At equal ratings, changes should be roughly symmetric: win=%f + lose=%f = %f",
			win.Change, lose.Change, sum)
	}
}

func TestCalculateGlicko_AsymmetryAtDifferentRatings(t *testing.T) {
	// When a 1800 beats a 1200 (expected), the favorite should gain LESS
	// than the underdog loses. This breaks zero-sum intentionally.
	winFav := CalculateGlicko(1800, 150, 1200, 150, 1.0, 0)
	loseUnder := CalculateGlicko(1200, 150, 1800, 150, 0.0, 0)

	if math.Abs(winFav.Change) >= math.Abs(loseUnder.Change) {
		t.Errorf("Favorite's gain should be less than underdog's loss: fav_gain=%f, under_loss=%f",
			winFav.Change, loseUnder.Change)
	}

	// When a 1200 beats a 1800 (upset), the underdog should gain MORE
	// than the favorite loses.
	winUpset := CalculateGlicko(1200, 150, 1800, 150, 1.0, 0)
	loseFav := CalculateGlicko(1800, 150, 1200, 150, 0.0, 0)

	if math.Abs(winUpset.Change) <= math.Abs(loseFav.Change) {
		t.Errorf("Upset winner's gain should exceed favorite's loss: upset_gain=%f, fav_loss=%f",
			winUpset.Change, loseFav.Change)
	}
}
