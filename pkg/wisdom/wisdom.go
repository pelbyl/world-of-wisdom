package wisdom

import (
	"math/rand"
	"sync"
	"time"
)

var quotes = []string{
	"The only true wisdom is in knowing you know nothing. - Socrates",
	"The fool doth think he is wise, but the wise man knows himself to be a fool. - William Shakespeare",
	"The secret of life, though, is to fall seven times and to get up eight times. - Paulo Coelho",
	"Any fool can know. The point is to understand. - Albert Einstein",
	"The only way to do great work is to love what you do. - Steve Jobs",
	"In the middle of difficulty lies opportunity. - Albert Einstein",
	"The journey of a thousand miles begins with one step. - Lao Tzu",
	"He who knows others is wise; he who knows himself is enlightened. - Lao Tzu",
	"The wise man does at once what the fool does finally. - Niccolo Machiavelli",
	"Knowledge speaks, but wisdom listens. - Jimi Hendrix",
	"The more I learn, the more I realize how much I don't know. - Albert Einstein",
	"Wisdom is not a product of schooling but of the lifelong attempt to acquire it. - Albert Einstein",
	"The greatest enemy of knowledge is not ignorance, it is the illusion of knowledge. - Stephen Hawking",
	"A wise man can learn more from a foolish question than a fool can learn from a wise answer. - Bruce Lee",
	"The wise man is one who knows what he does not know. - Lao Tzu",
	"Yesterday I was clever, so I wanted to change the world. Today I am wise, so I am changing myself. - Rumi",
	"The measure of intelligence is the ability to change. - Albert Einstein",
	"Turn your wounds into wisdom. - Oprah Winfrey",
	"Wisdom comes from experience. Experience is often a result of lack of wisdom. - Terry Pratchett",
	"The beginning of wisdom is to desire it. - Solomon Ibn Gabirol",
	"Patience is the companion of wisdom. - Saint Augustine",
	"The wise are instructed by reason, average minds by experience, the stupid by necessity and the brute by instinct. - Marcus Tullius Cicero",
	"Knowing yourself is the beginning of all wisdom. - Aristotle",
	"The invariable mark of wisdom is to see the miraculous in the common. - Ralph Waldo Emerson",
	"Wisdom begins in wonder. - Socrates",
}

type QuoteProvider struct {
	quotes []string
	mu     sync.RWMutex
	rng    *rand.Rand
}

func NewQuoteProvider() *QuoteProvider {
	return &QuoteProvider{
		quotes: quotes,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (qp *QuoteProvider) GetRandomQuote() string {
	qp.mu.RLock()
	defer qp.mu.RUnlock()
	
	if len(qp.quotes) == 0 {
		return "No wisdom available at this time."
	}
	
	index := qp.rng.Intn(len(qp.quotes))
	return qp.quotes[index]
}

func (qp *QuoteProvider) AddQuote(quote string) {
	qp.mu.Lock()
	defer qp.mu.Unlock()
	
	qp.quotes = append(qp.quotes, quote)
}

func (qp *QuoteProvider) GetQuoteCount() int {
	qp.mu.RLock()
	defer qp.mu.RUnlock()
	
	return len(qp.quotes)
}