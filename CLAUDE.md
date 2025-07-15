# Word of Wisdom Improvements Guide

## Current System Limitations:

- Global difficulty: All clients get the same difficulty level
- No per-IP tracking: Difficulty doesn't consider individual client behavior
- Simple metrics: Only tracks average solve time and connection rate globally

## Proposed Solution: Per-Client Adaptive Difficulty

1. Behavioral Scoring System

Track each client/IP and assign difficulty based on their behavior:

type ClientBehavior struct {
    IP              string
    ConnectionCount int
    FailureRate     float64
    AvgSolveTime    time.Duration
    LastConnection  time.Time
    ReconnectRate   float64
    Difficulty      int
}

2. Difficulty Calculation Per Client

Different factors would contribute to a client's difficulty:

   - Connection Frequency: High frequency = higher difficulty
   - Failure Rate: Many failed attempts = higher difficulty
   - Solve Time: Suspiciously fast solving = higher difficulty
   - Reconnect Pattern: Rapid reconnects = higher difficulty
   - Time Since Last Visit: Returning after long time = lower difficulty

1. Implementation Strategy

Would you like me to implement this per-client difficulty system? The key changes would be:

1. Add per-IP tracking table to database
2. Modify challenge generation to use client-specific difficulty
3. Add behavioral analysis to detect DDoS patterns
4. Implement reputation decay so good behavior over time reduces difficulty

This would allow the system to:

   - Keep normal clients at low difficulty (1-2)
   - Progressively increase difficulty for aggressive clients (up to 6)
   - Automatically detect and penalize DDoS behavior
   - Maintain good user experience for legitimate users

## Monitoring

Implement UI system which show list of active clients and their difficulty, also mark normal clients with green color and aggressive clients with red color.

In active connections list, show the following information for each client:

- IP address
- Difficulty
- Connection count
- Failure rate
- Solve time
- Reconnect rate
- Time since last visit

Add dashboard to show clients hit high difficulty, also show the list of clients which hit high difficulty.

Add the description of this solution to the README.md file
