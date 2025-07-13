# Word of Wisdom TCP Server - Project Improvement Plan

For the existing project please improve it as follows.

1. Replace SHA-256 with Scrypt/Argon2 for enhanced security.
2. Set up PostgreSQL/TimescaleDB for metrics and application data
3. Examine the fromntend, I need to store all the statistics and dont lose it when i refresh the page. I need more interactive reaction when I click Demo mode and etc. Also there is an error: when one Extreme session is over, and it shows some error with web socket and I cant start a new session since buttons doesn't work.
4. Create REST API gateway for frontend-backend communication. I need sqlc generated code.
5. Split into microservices: TCP server, API server, frontend.
6. Configure Nginx load balancer and reverse proxy.
7. Set up GitHub Actions for automated deployment.
8. Deploy across multiple VPS instances. I'm planning to run main server docekr container on VPS with public IP address, please make sure that the server is running and accessible from the internet. I have ssh access to two servers(ssh vps_1_1 and ssh vps_1_2), maybe we can configure gilhub action with simple deploy? If its too comlicated we can do smth like "make run-server" instruction to run the project on vps.
9. Add tools for testing the service in more realistic scenario
I mean I run many docker containers locally simulate many clients, and this cluster of clients try to connect to the server and solve the puzzle.
