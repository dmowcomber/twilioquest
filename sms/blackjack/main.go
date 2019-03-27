package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type state struct {
	money      int
	betAmount  int
	dealerHand []card
	playerHand []card
	deck       []card
	handCount  int
}

type card struct {
	suit string
	face string
}

func (c card) String() string {
	return fmt.Sprintf("%s", c.face)
}

// func (c []card) String() string {
// 	return fmt.Sprintf("%s", c.face)
// }

var (
	// TODO: cache it locally??
	states         = map[string]*state{}
	cardFaceValues = map[string]int{
		"A":  0, // handled in a function...
		"2":  2,
		"3":  3,
		"4":  4,
		"5":  5,
		"6":  6,
		"7":  7,
		"8":  8,
		"9":  9,
		"10": 10,
		"J":  10,
		"Q":  10,
		"K":  10,
	}
	suits = []string{
		"hearts",
		"diamonds",
		"spades",
		"clubs",
	}
)

func createShuffledDeck() []card {
	deck := make([]card, 52)
	var deckIndex int
	for _, suit := range suits {
		for face := range cardFaceValues {
			deck[deckIndex] = newCard(suit, face)
			deckIndex++
		}
	}
	// shuffle
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })

	return deck
}

func newCard(suit string, face string) card {
	return card{
		suit: suit,
		face: face,
	}
}

func newState() *state {
	s := &state{
		money:     100,
		betAmount: 20,
		deck:      createShuffledDeck(),
	}
	s.dealNewHands()
	return s
}

func (s *state) dealNewHands() bool {
	var newShuffle bool
	s.handCount++
	if s.handCount > 4 {
		newShuffle = true
		s.deck = createShuffledDeck()
		s.handCount = 1
	}

	s.dealerHand = make([]card, 0, 10)
	s.playerHand = make([]card, 0, 10)
	s.drawPlayer()
	s.drawDealer()
	s.drawPlayer()
	s.drawDealer()

	return newShuffle
}

func (s *state) drawDealer() {
	var c card
	// TODO mutex!
	if len(s.deck) == 0 {
		return
	}
	c, s.deck = s.deck[len(s.deck)-1], s.deck[:len(s.deck)-1]
	s.dealerHand = append(s.dealerHand, c)
}

func (s *state) drawPlayer() {
	var c card

	// TODO mutex
	if len(s.deck) == 0 {
		return
	}
	c, s.deck = s.deck[len(s.deck)-1], s.deck[:len(s.deck)-1]
	s.playerHand = append(s.playerHand, c)
}

func main() {
	http.HandleFunc("/sms", smsEndpoint)
	fmt.Println("listening on port 8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

var s *state

func smsEndpoint(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic", r)
			w.Write([]byte(lazyTwimlMessage("internal error")))
		}
	}()

	textMessage := r.URL.Query().Get("Body")
	textMessageLower := strings.ToLower(textMessage)
	fmt.Printf("request body: %q\n", textMessage)

	from := r.URL.Query().Get("From")

	s, ok := states[from]
	if ok {
		if len(s.deck) == 0 {
			s.dealNewHands()
			message := "no more cards. this should never happen... shuffling"
			message = message + "\n\nDealing new hands\n" + s.getStatus()
			w.Write([]byte(lazyTwimlMessage(message)))

			return
		}

		if strings.HasPrefix(textMessageLower, "hit") || textMessageLower == "h" {
			s.drawPlayer()

			playerHandScore, _ := getHandScore(s.playerHand)
			if playerHandScore > 21 {
				message := fmt.Sprintf("you bust with %d: %s\n", playerHandScore, formatHand(s.playerHand))
				wasShuffled := s.dealNewHands()
				if wasShuffled {
					message = message + "\n\nShuffling deck\n"
				}
				// TODO: can this be in a separate message??
				message = message + "\n\nDealing new hands\n" + s.getStatus()

				w.Write([]byte(lazyTwimlMessage(message)))

				return
			}
			w.Write([]byte(lazyTwimlMessage(s.getStatus())))
		} else if strings.HasPrefix(textMessageLower, "stay") || textMessageLower == "s" {

			s.dealerPlay()

			message := fmt.Sprintf(s.getStayEndStatus())
			wasShuffled := s.dealNewHands()
			if wasShuffled {
				message = message + "\n\nShuffling deck\n"
			}
			// TODO: can this be in a separate message??
			message = message + "\n\nDealing new hands\n" + s.getStatus()
			w.Write([]byte(lazyTwimlMessage(message)))
			return
		} else {
			w.Write([]byte(lazyTwimlMessage("do you want to hit or stay?")))
			return
		}
		return
	}

	// mutex states
	s = newState()
	states[from] = s

	message := "Welcome to Blackjack!\nShuffling Cards...\n\n" + s.getStatus()
	w.Write([]byte(lazyTwimlMessage(message)))
	return

	// TODO mutex on state check
}

func (s *state) getStatus() string {
	if len(s.dealerHand) < 2 {
		// TODO: new state
		return "internal error"
	}

	playerHandScore, softScore := getHandScore(s.playerHand)
	score := fmt.Sprintf("%d", playerHandScore)
	if playerHandScore != softScore {
		score = fmt.Sprintf("%d or %d", playerHandScore, softScore)
	}
	return fmt.Sprintf("dealer's face up card: %s\nyour hand: %v\nyour score: %s\nType: hit or stay", s.dealerHand[0], formatHand(s.playerHand), score)
}

func (s *state) getStayEndStatus() string {
	dealerScore, _ := getHandScore(s.dealerHand)
	playerScore, _ := getHandScore(s.playerHand)

	winMessage := "Push. You tied with the dealer."
	if dealerScore > 21 {
		winMessage = "Dealer bust. You won!"
	} else if dealerScore > playerScore {
		winMessage = "You lost."
	} else if playerScore > dealerScore {
		winMessage = "You won!"
	}
	winMessage = fmt.Sprintf("%s\nYou had %d and the dealer had %d.", winMessage, playerScore, dealerScore)
	return fmt.Sprintf("dealer's hand: %v\nyour hand: %v\n%s\n", formatHand(s.dealerHand), formatHand(s.playerHand), winMessage)
}

func (s *state) dealerPlay() {
	for {
		score, softScore := getHandScore(s.dealerHand)
		fmt.Printf("dealer score: %d or %d\n", score, softScore)

		if score < 17 {
			s.drawDealer()
		} else {
			return
		}
	}
}

func formatHand(cards []card) string {
	s := make([]string, 0, len(cards))
	for _, c := range cards {
		s = append(s, c.face)
	}
	return strings.Join(s, ", ")
}

func doubleBet() {}
func resetBet()  {}
func maxBet()    {}

func getHandScore(cards []card) (score int, softScore int) {
	// add non-aces
	for _, c := range cards {
		if c.face == "A" {
			continue
		}
		score = score + cardFaceValues[c.face]
	}
	softScore = score

	// add aces
	var aceCount int
	for _, c := range cards {
		if c.face != "A" {
			continue
		}

		aceCount++
		score = score + 11
		softScore = softScore + 1
	}

	// convert ace 11s to 1s
	for i := 0; i < aceCount; i++ {
		if score > 21 {
			score = score - 10
		}
	}
	return score, softScore
}

func lazyTwimlMessage(message string) string {

	// TODO: resp should have remaining money, hands played, last winning
	// split?????

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Message>
        %s
    </Message>
</Response>`, message)
}
