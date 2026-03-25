// Package main provides the Compare server — a head-to-head comparison
// platform with Glicko-inspired ratings.
//
// This file implements the rating system, based on Mark Glickman's Glicko
// algorithm with an additional head-to-head dampening factor.
//
// Key concepts:
//   - Rating: a player's estimated strength (default 1500).
//   - RD (Rating Deviation): uncertainty in the rating. High RD means the
//     system is unsure; low RD means it is confident. Starts at 350 for
//     new items and decreases toward 50 as matches accumulate.
//   - g(RD): a weighting function that reduces the influence of an
//     opponent whose rating is uncertain.
//   - E (expected score): the probability of winning, adjusted by g(RD).
//   - H2H dampening: repeated matchups between the same pair yield
//     progressively smaller rating updates, preventing two items from
//     endlessly trading points.
package main

import "math"

// Rating system constants.
const (
	glickoQ  = 0.0057565 // ln(10) / 400
	minRD    = 50.0       // Floor: maximum confidence
	maxRD    = 350.0      // Ceiling: minimum confidence (new items)
	eloFloor = 100.0      // No rating may drop below this value
)

// RatingResult holds the outcome of a single rating calculation.
type RatingResult struct {
	NewRating float64 // Updated rating after the match
	NewRD     float64 // Updated rating deviation
	Change    float64 // Signed rating change applied
}

// GlickoG computes the Glicko g(RD) weighting function.
// It reduces the influence of opponents whose rating is uncertain (high RD).
func GlickoG(rd float64) float64 {
	return 1.0 / math.Sqrt(1.0+3.0*glickoQ*glickoQ*rd*rd/(math.Pi*math.Pi))
}

// GlickoExpected computes the expected score E(rating, oppRating, oppRD).
// Returns a value in [0, 1] representing the probability that the player
// with the given rating beats an opponent with oppRating and oppRD.
func GlickoExpected(rating, oppRating, oppRD float64) float64 {
	return 1.0 / (1.0 + math.Pow(10, -GlickoG(oppRD)*(rating-oppRating)/400.0))
}

// glickoDSquared computes d², the estimation variance used to update RD.
func glickoDSquared(rating, oppRating, oppRD float64) float64 {
	g := GlickoG(oppRD)
	e := GlickoExpected(rating, oppRating, oppRD)
	return 1.0 / (glickoQ * glickoQ * g * g * e * (1.0 - e))
}

// CalculateGlicko computes the new rating and RD for one player after a
// single match.
//
// Parameters:
//   - rating, rd: the player's current rating and deviation.
//   - oppRating, oppRD: the opponent's current rating and deviation.
//   - score: 1.0 for a win, 0.0 for a loss.
//   - h2hCount: number of prior head-to-head matches between these items.
//     Used to apply diminishing returns on repeated matchups.
func CalculateGlicko(rating, rd, oppRating, oppRD, score float64, h2hCount int) RatingResult {
	g := GlickoG(oppRD)
	e := GlickoExpected(rating, oppRating, oppRD)
	dSq := glickoDSquared(rating, oppRating, oppRD)

	// Head-to-head dampening: first match has full weight, subsequent
	// matches decay on a 1/(1 + 0.15*n) curve.
	h2hFactor := 1.0 / (1.0 + 0.15*float64(h2hCount))

	// Rating update (Glicko formula with h2h dampening)
	ratingChange := h2hFactor * (glickoQ / (1.0/rd/rd + 1.0/dSq)) * g * (score - e)

	// Asymmetric scaling: break the zero-sum symmetry so that gain and loss
	// reflect how "expected" the outcome was for each player individually.
	//
	// The player's own expected score (e) tells us their position:
	//   - Favorite (e ≈ 0.95): expected to win
	//   - Underdog (e ≈ 0.05): expected to lose
	//
	// We want:
	//   Favorite wins  → small gain    (scale down by e:    0.95 * change is small because base (1-e) is already small)
	//   Favorite loses → large penalty (scale up by e:      0.95 * |change| amplifies the loss)
	//   Underdog wins  → large gain    (scale up by 1-e:    0.95 * change boosts the gain)
	//   Underdog loses → small penalty (scale down by 1-e:  but (1-e)≈0.05 here is wrong direction)
	//
	// Solution: scale by how surprising the result was for THIS player.
	//   Win:  multiply by 2*(1-e) — surprising wins (low e) get boosted
	//   Loss: multiply by 2*(1-e) — expected losses (low e) get cushioned
	// This means each side of the SAME match gets a different multiplier.
	ratingChange *= 2 * (1 - e)

	newRating := rating + ratingChange

	// RD update: shrinks toward minRD with each match
	newRD := math.Sqrt(1.0 / (1.0/rd/rd + 1.0/dSq))
	newRD = math.Max(minRD, math.Min(maxRD, newRD))

	// Enforce rating floor
	if newRating < eloFloor {
		ratingChange = eloFloor - rating
		newRating = eloFloor
	}

	return RatingResult{
		NewRating: newRating,
		NewRD:     newRD,
		Change:    ratingChange,
	}
}
