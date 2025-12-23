// ============================================
// IMPOSTER GAME - Client Application
// ============================================

(function() {
    'use strict';

    // ============================================
    // State Management
    // ============================================
    const state = {
        playerId: null,
        roomCode: null,
        nickname: null,
        isHost: false,
        phase: 'LOBBY',
        players: [],
        role: null,
        secretWord: null,
        submissions: [],
        currentPlayerId: null,
        hasVoted: false,
        votingSeconds: 20,
        ws: null
    };

    // ============================================
    // DOM Elements
    // ============================================
    const screens = {
        home: document.getElementById('screen-home'),
        lobby: document.getElementById('screen-lobby'),
        role: document.getElementById('screen-role'),
        submission: document.getElementById('screen-submission'),
        voting: document.getElementById('screen-voting'),
        results: document.getElementById('screen-results')
    };

    const elements = {
        // Home
        btnCreate: document.getElementById('btn-create'),
        inputRoomCode: document.getElementById('input-room-code'),
        btnJoin: document.getElementById('btn-join'),
        stats: document.getElementById('stats'),

        // Lobby
        roomCode: document.getElementById('room-code'),
        btnCopyLink: document.getElementById('btn-copy-link'),
        nicknameForm: document.getElementById('nickname-form'),
        inputNickname: document.getElementById('input-nickname'),
        btnSetNickname: document.getElementById('btn-set-nickname'),
        playersSection: document.getElementById('players-section'),
        playerCount: document.getElementById('player-count'),
        playersGrid: document.getElementById('players-grid'),
        hostControls: document.getElementById('host-controls'),
        btnStart: document.getElementById('btn-start'),
        startHint: document.getElementById('start-hint'),
        waitingMessage: document.getElementById('waiting-message'),

        // Role
        roleCard: document.getElementById('role-card'),
        roleName: document.getElementById('role-name'),
        secretWordContainer: document.getElementById('secret-word-container'),
        secretWord: document.getElementById('secret-word'),
        imposterMessage: document.getElementById('imposter-message'),

        // Submission
        currentPlayerName: document.getElementById('current-player-name'),
        submissionsList: document.getElementById('submissions-list'),
        yourTurnForm: document.getElementById('your-turn-form'),
        inputWord: document.getElementById('input-word'),
        btnSubmitWord: document.getElementById('btn-submit-word'),
        waitingTurn: document.getElementById('waiting-turn'),
        waitingForPlayer: document.getElementById('waiting-for-player'),

        // Voting
        countdownNumber: document.getElementById('countdown-number'),
        votesCast: document.getElementById('votes-cast'),
        votesTotal: document.getElementById('votes-total'),
        votingSubmissionsList: document.getElementById('voting-submissions-list'),
        votingGrid: document.getElementById('voting-grid'),
        votedMessage: document.getElementById('voted-message'),

        // Results
        winnerBanner: document.getElementById('winner-banner'),
        winnerText: document.getElementById('winner-text'),
        imposterName: document.getElementById('imposter-name'),
        revealedWord: document.getElementById('revealed-word'),
        votesBreakdown: document.getElementById('votes-breakdown'),
        playAgainControls: document.getElementById('play-again-controls'),
        btnPlayAgain: document.getElementById('btn-play-again'),
        waitingNewRound: document.getElementById('waiting-new-round'),

        // Toast
        toastContainer: document.getElementById('toast-container')
    };

    // ============================================
    // Screen Management
    // ============================================
    function showScreen(screenName) {
        Object.values(screens).forEach(screen => screen.classList.remove('active'));
        if (screens[screenName]) {
            screens[screenName].classList.add('active');
        }
    }

    // ============================================
    // Toast Notifications
    // ============================================
    function showToast(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        elements.toastContainer.appendChild(toast);

        setTimeout(() => {
            toast.remove();
        }, 3000);
    }

    // ============================================
    // API Functions
    // ============================================
    async function createRoom() {
        try {
            const response = await fetch('/api/rooms', { method: 'POST' });
            const data = await response.json();

            if (data.success) {
                state.roomCode = data.data.roomCode;
                joinRoom(data.data.roomCode);
            } else {
                showToast(data.error.message, 'error');
            }
        } catch (error) {
            showToast('Failed to create room', 'error');
            console.error('Create room error:', error);
        }
    }

    async function checkRoom(roomCode) {
        try {
            const response = await fetch(`/api/rooms/${roomCode}`);
            const data = await response.json();
            return data.success ? data.data : null;
        } catch (error) {
            console.error('Check room error:', error);
            return null;
        }
    }

    function joinRoom(roomCode) {
        state.roomCode = roomCode.toUpperCase();
        elements.roomCode.textContent = state.roomCode;
        showScreen('lobby');
        elements.inputNickname.focus();
    }

    // ============================================
    // WebSocket Functions
    // ============================================
    function connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws?roomCode=${state.roomCode}` +
            (state.playerId ? `&playerId=${state.playerId}` : '');

        state.ws = new WebSocket(wsUrl);

        state.ws.onopen = () => {
            console.log('WebSocket connected');
        };

        state.ws.onmessage = (event) => {
            // Handle multiple messages (they can be newline-separated)
            const messages = event.data.split('\n');
            messages.forEach(msgStr => {
                if (msgStr.trim()) {
                    try {
                        const message = JSON.parse(msgStr);
                        handleMessage(message);
                    } catch (e) {
                        console.error('Failed to parse message:', e);
                    }
                }
            });
        };

        state.ws.onclose = () => {
            console.log('WebSocket disconnected');
            // Try to reconnect after a delay
            setTimeout(() => {
                if (state.roomCode && state.playerId) {
                    connectWebSocket();
                }
            }, 3000);
        };

        state.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    function sendMessage(type, payload = {}) {
        if (state.ws && state.ws.readyState === WebSocket.OPEN) {
            state.ws.send(JSON.stringify({ type, payload }));
        }
    }

    function handleMessage(message) {
        console.log('Received:', message.type, message.payload);

        switch (message.type) {
            case 'connected':
                handleConnected(message.payload);
                break;
            case 'error':
                handleError(message.payload);
                break;
            case 'lobby_update':
            case 'PLAYER_JOINED':
            case 'PLAYER_LEFT':
            case 'PLAYER_RECONNECTED':
                handleLobbyUpdate(message.payload);
                break;
            case 'ROLES_ASSIGNED':
                handleRoleAssigned(message.payload);
                break;
            case 'SUBMISSION_MADE':
                handleSubmissionUpdate(message.payload);
                break;
            case 'VOTING_STARTED':
                handleVotingStarted(message.payload);
                break;
            case 'VOTE_CAST':
                handleVoteUpdate(message.payload);
                break;
            case 'ROUND_ENDED':
                handleRoundResults(message.payload);
                break;
            case 'pong':
                // Heartbeat response
                break;
            default:
                console.log('Unknown message type:', message.type);
        }
    }

    function handleConnected(payload) {
        state.playerId = payload.playerId;
        state.roomCode = payload.gameId;

        // Restore state from gameState
        if (payload.gameState) {
            const gs = payload.gameState;
            state.phase = gs.phase;
            state.isHost = gs.hostId === state.playerId;
            
            if (gs.players) {
                state.players = gs.players;
            }
            if (gs.role) {
                state.role = gs.role;
            }
            if (gs.secretWord) {
                state.secretWord = gs.secretWord;
            }

            // Navigate to appropriate screen based on phase
            switch (gs.phase) {
                case 'LOBBY':
                    updateLobbyUI();
                    break;
                case 'ROLE_ASSIGNMENT':
                    if (gs.role) {
                        showRoleScreen(gs.role, gs.secretWord);
                    }
                    break;
                case 'SUBMISSION':
                    state.submissions = gs.submissions || [];
                    state.currentPlayerId = gs.currentPlayerId;
                    showSubmissionScreen();
                    break;
                case 'VOTING':
                    showVotingScreen();
                    break;
                case 'RESULTS':
                    if (gs.results) {
                        showResultsScreen(gs.results, gs.winner, gs.imposterId, gs.secretWord);
                    }
                    break;
            }
        }

        // Save player ID to localStorage for reconnection
        localStorage.setItem(`imposter_player_${state.roomCode}`, state.playerId);
    }

    function handleError(payload) {
        showToast(payload.message, 'error');
    }

    function handleLobbyUpdate(payload) {
        if (payload.players) {
            state.players = payload.players;
        }
        if (payload.hostId) {
            state.isHost = payload.hostId === state.playerId;
        }
        updateLobbyUI();
    }

    function handleRoleAssigned(payload) {
        state.role = payload.role;
        state.secretWord = payload.secretWord || null;
        state.phase = 'ROLE_ASSIGNMENT';
        showRoleScreen(payload.role, payload.secretWord);
    }

    function handleSubmissionUpdate(payload) {
        state.phase = 'SUBMISSION';
        if (payload.submissions) {
            state.submissions = payload.submissions;
        }
        if (payload.currentPlayerId !== undefined) {
            state.currentPlayerId = payload.currentPlayerId;
        }

        // Check if we're already on submission screen
        if (screens.submission.classList.contains('active')) {
            updateSubmissionUI();
        } else {
            showSubmissionScreen();
        }

        // Check if all submitted -> voting will start
        if (payload.isComplete) {
            // Voting phase will be triggered by VOTING_STARTED
        }
    }

    function handleVotingStarted(payload) {
        state.phase = 'VOTING';
        state.hasVoted = false;
        state.votingSeconds = payload.remainingSeconds || 20;
        if (payload.players) {
            state.players = payload.players;
        }
        showVotingScreen();
    }

    function handleVoteUpdate(payload) {
        if (payload.remainingSeconds !== undefined) {
            updateCountdown(payload.remainingSeconds);
        }
        if (payload.votedCount !== undefined) {
            elements.votesCast.textContent = payload.votedCount;
            elements.votesTotal.textContent = payload.totalPlayers;
        }
    }

    function handleRoundResults(payload) {
        state.phase = 'RESULTS';
        showResultsScreen(payload.votes, payload.winner, payload.imposterId, payload.secretWord);
    }

    // ============================================
    // UI Update Functions
    // ============================================
    function updateLobbyUI() {
        // Update player grid
        elements.playersGrid.innerHTML = '';
        state.players.forEach(player => {
            const card = document.createElement('div');
            card.className = 'player-card';
            if (player.id === state.playerId) {
                card.classList.add('is-you');
            }
            if (state.isHost && player.id === state.playerId) {
                // Find if this player is host (we need to check from state)
            }
            // Check if this player is the host
            const isPlayerHost = state.players.length > 0 && 
                state.players.find(p => p.id === player.id) && 
                elements.hostControls.style.display !== 'none' && 
                player.id === state.playerId && state.isHost;

            if (player.status === 'DISCONNECTED') {
                card.classList.add('disconnected');
            }

            card.innerHTML = `<div class="player-nickname">${escapeHtml(player.nickname)}</div>`;
            elements.playersGrid.appendChild(card);
        });

        // Update player count
        elements.playerCount.textContent = `${state.players.length}/10`;

        // Update host controls
        if (state.isHost) {
            elements.hostControls.style.display = 'block';
            elements.waitingMessage.style.display = 'none';

            const canStart = state.players.length >= 4;
            elements.btnStart.disabled = !canStart;
            elements.startHint.textContent = canStart 
                ? 'Ready to start!' 
                : `Need ${4 - state.players.length} more player(s)`;
        } else {
            elements.hostControls.style.display = 'none';
            elements.waitingMessage.style.display = 'block';
        }
    }

    function showRoleScreen(role, secretWord) {
        elements.roleName.textContent = role;
        elements.roleName.className = 'role-name ' + role.toLowerCase();

        if (role === 'VILEK') {
            elements.secretWordContainer.style.display = 'block';
            elements.secretWord.textContent = secretWord;
            elements.imposterMessage.style.display = 'none';
        } else {
            elements.secretWordContainer.style.display = 'none';
            elements.imposterMessage.style.display = 'block';
        }

        showScreen('role');

        // Auto-transition to submission after delay (handled by server)
    }

    function showSubmissionScreen() {
        showScreen('submission');
        updateSubmissionUI();
    }

    function updateSubmissionUI() {
        // Update current player
        const currentPlayer = state.players.find(p => p.id === state.currentPlayerId);
        elements.currentPlayerName.textContent = currentPlayer 
            ? currentPlayer.nickname 
            : 'Waiting...';

        // Update submissions list
        elements.submissionsList.innerHTML = '';
        state.submissions.forEach(sub => {
            const item = document.createElement('div');
            item.className = 'submission-item';
            item.innerHTML = `
                <span class="submission-order">${sub.order}.</span>
                <span class="submission-player">${escapeHtml(sub.nickname)}</span>
                <span class="submission-word">${escapeHtml(sub.word)}</span>
            `;
            elements.submissionsList.appendChild(item);
        });

        // Show/hide turn form
        const isMyTurn = state.currentPlayerId === state.playerId;
        elements.yourTurnForm.style.display = isMyTurn ? 'block' : 'none';
        elements.waitingTurn.style.display = isMyTurn ? 'none' : 'block';

        if (isMyTurn) {
            elements.inputWord.value = '';
            elements.inputWord.focus();
        } else {
            const waitingFor = state.players.find(p => p.id === state.currentPlayerId);
            elements.waitingForPlayer.textContent = waitingFor 
                ? waitingFor.nickname 
                : 'next player';
        }
    }

    function showVotingScreen() {
        showScreen('voting');
        state.hasVoted = false;

        // Reset vote UI
        elements.votedMessage.style.display = 'none';
        updateCountdown(state.votingSeconds);

        // Build submissions list for reference
        elements.votingSubmissionsList.innerHTML = '';
        state.submissions.forEach(sub => {
            const item = document.createElement('div');
            item.className = 'voting-submission-item';
            item.dataset.playerId = sub.playerId;
            item.innerHTML = `
                <span class="player-name">${escapeHtml(sub.nickname)}</span>
                <span class="word">${escapeHtml(sub.word)}</span>
            `;
            
            // Highlight on hover to help identify
            item.addEventListener('mouseenter', () => {
                highlightPlayer(sub.playerId, true);
            });
            item.addEventListener('mouseleave', () => {
                highlightPlayer(sub.playerId, false);
            });
            
            elements.votingSubmissionsList.appendChild(item);
        });

        // Build voting grid
        elements.votingGrid.innerHTML = '';
        state.players.forEach(player => {
            const card = document.createElement('div');
            card.className = 'vote-card';
            card.dataset.playerId = player.id;
            
            if (player.id === state.playerId) {
                card.classList.add('is-you');
            }

            card.innerHTML = `<div class="vote-card-name">${escapeHtml(player.nickname)}</div>`;
            
            card.addEventListener('click', () => {
                if (!state.hasVoted && player.id !== state.playerId) {
                    castVote(player.id);
                    
                    // Mark as selected
                    document.querySelectorAll('.vote-card').forEach(c => c.classList.remove('selected'));
                    card.classList.add('selected');
                }
            });
            
            // Highlight on hover to help identify in submissions
            card.addEventListener('mouseenter', () => {
                highlightSubmission(player.id, true);
            });
            card.addEventListener('mouseleave', () => {
                highlightSubmission(player.id, false);
            });

            elements.votingGrid.appendChild(card);
        });
    }
    
    function highlightPlayer(playerId, highlight) {
        const card = elements.votingGrid.querySelector(`[data-player-id="${playerId}"]`);
        if (card) {
            card.classList.toggle('highlighted', highlight);
        }
    }
    
    function highlightSubmission(playerId, highlight) {
        const item = elements.votingSubmissionsList.querySelector(`[data-player-id="${playerId}"]`);
        if (item) {
            item.classList.toggle('highlighted', highlight);
        }
    }

    function updateCountdown(seconds) {
        elements.countdownNumber.textContent = seconds;
        elements.countdownNumber.classList.toggle('urgent', seconds <= 5);
    }

    function castVote(targetId) {
        sendMessage('cast_vote', { targetPlayerId: targetId });
        state.hasVoted = true;
        elements.votedMessage.style.display = 'block';
        
        // Disable all vote cards
        document.querySelectorAll('.vote-card').forEach(card => {
            card.classList.add('disabled');
        });
    }

    function showResultsScreen(votes, winner, imposterId, secretWord) {
        showScreen('results');

        // Winner banner
        const isVileksWin = winner === 'VILEK';
        elements.winnerBanner.className = 'winner-banner ' + (isVileksWin ? 'vileks-win' : 'imposter-wins');
        elements.winnerText.textContent = isVileksWin ? 'VILEKS WIN!' : 'IMPOSTER WINS!';

        // Imposter reveal
        const imposter = state.players.find(p => p.id === imposterId);
        elements.imposterName.textContent = imposter ? imposter.nickname : 'Unknown';

        // Secret word
        elements.revealedWord.textContent = secretWord;

        // Votes breakdown
        elements.votesBreakdown.innerHTML = '<h4>VOTE BREAKDOWN</h4>';
        
        // Sort by vote count
        const sortedVotes = [...votes].sort((a, b) => b.voteCount - a.voteCount);
        
        sortedVotes.forEach(vote => {
            const result = document.createElement('div');
            result.className = 'vote-result';
            if (vote.isImposter) {
                result.classList.add('is-imposter');
            }
            
            result.innerHTML = `
                <div>
                    <div class="vote-result-name">
                        ${escapeHtml(vote.nickname)}
                        ${vote.isImposter ? ' 🎭' : ''}
                    </div>
                    ${vote.votedBy && vote.votedBy.length > 0 
                        ? `<div class="vote-result-voters">Voted by: ${vote.votedBy.map(n => escapeHtml(n)).join(', ')}</div>` 
                        : ''}
                </div>
                <div class="vote-result-count">${vote.voteCount}</div>
            `;
            
            elements.votesBreakdown.appendChild(result);
        });

        // Play again controls
        if (state.isHost) {
            elements.playAgainControls.style.display = 'block';
            elements.waitingNewRound.style.display = 'none';
        } else {
            elements.playAgainControls.style.display = 'none';
            elements.waitingNewRound.style.display = 'block';
        }
    }

    // ============================================
    // Utility Functions
    // ============================================
    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    function copyToClipboard(text) {
        if (navigator.clipboard) {
            navigator.clipboard.writeText(text)
                .then(() => showToast('Copied to clipboard!', 'success'))
                .catch(() => showToast('Failed to copy', 'error'));
        } else {
            // Fallback
            const textarea = document.createElement('textarea');
            textarea.value = text;
            document.body.appendChild(textarea);
            textarea.select();
            try {
                document.execCommand('copy');
                showToast('Copied to clipboard!', 'success');
            } catch (e) {
                showToast('Failed to copy', 'error');
            }
            document.body.removeChild(textarea);
        }
    }

    // ============================================
    // Event Listeners
    // ============================================
    function setupEventListeners() {
        // Home screen
        elements.btnCreate.addEventListener('click', createRoom);

        elements.btnJoin.addEventListener('click', async () => {
            const code = elements.inputRoomCode.value.trim().toUpperCase();
            if (code.length < 4) {
                showToast('Please enter a valid room code', 'error');
                return;
            }

            const room = await checkRoom(code);
            if (room) {
                if (room.canJoin) {
                    joinRoom(code);
                } else {
                    showToast('Cannot join this room (game in progress or full)', 'error');
                }
            } else {
                showToast('Room not found', 'error');
            }
        });

        elements.inputRoomCode.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                elements.btnJoin.click();
            }
        });

        // Lobby screen
        elements.btnCopyLink.addEventListener('click', () => {
            const link = `${window.location.origin}/join/${state.roomCode}`;
            copyToClipboard(link);
        });

        elements.btnSetNickname.addEventListener('click', () => {
            const nickname = elements.inputNickname.value.trim();
            if (nickname.length < 1) {
                showToast('Please enter a nickname', 'error');
                return;
            }
            if (nickname.length > 15) {
                showToast('Nickname too long (max 15 chars)', 'error');
                return;
            }

            state.nickname = nickname;
            
            // Try to restore player ID from localStorage
            const savedPlayerId = localStorage.getItem(`imposter_player_${state.roomCode}`);
            if (savedPlayerId) {
                state.playerId = savedPlayerId;
            }

            // Connect WebSocket
            connectWebSocket();

            // Wait for connection then send join
            const checkAndJoin = () => {
                if (state.ws && state.ws.readyState === WebSocket.OPEN) {
                    sendMessage('join_lobby', { nickname });
                    elements.nicknameForm.style.display = 'none';
                    elements.playersSection.style.display = 'block';
                } else {
                    setTimeout(checkAndJoin, 100);
                }
            };
            checkAndJoin();
        });

        elements.inputNickname.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                elements.btnSetNickname.click();
            }
        });

        elements.btnStart.addEventListener('click', () => {
            sendMessage('start_game');
        });

        // Submission screen
        elements.btnSubmitWord.addEventListener('click', () => {
            const word = elements.inputWord.value.trim();
            if (word.length < 1) {
                showToast('Please enter a word', 'error');
                return;
            }
            if (word.includes(' ')) {
                showToast('Only one word allowed (no spaces)', 'error');
                return;
            }

            sendMessage('submit_word', { word });
            elements.inputWord.value = '';
        });

        elements.inputWord.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                elements.btnSubmitWord.click();
            }
        });

        // Results screen
        elements.btnPlayAgain.addEventListener('click', () => {
            sendMessage('request_new_round');
        });

        // Heartbeat
        setInterval(() => {
            if (state.ws && state.ws.readyState === WebSocket.OPEN) {
                sendMessage('ping');
            }
        }, 30000);
    }

    // ============================================
    // Routing (handle /join/:roomCode URLs)
    // ============================================
    function handleRouting() {
        const path = window.location.pathname;
        const joinMatch = path.match(/^\/join\/([A-Za-z0-9]+)$/);

        if (joinMatch) {
            const roomCode = joinMatch[1].toUpperCase();
            checkRoom(roomCode).then(room => {
                if (room) {
                    joinRoom(roomCode);
                } else {
                    showToast('Room not found', 'error');
                    // Redirect to home
                    window.history.pushState({}, '', '/');
                }
            });
        }
    }

    // ============================================
    // Load Stats
    // ============================================
    async function loadStats() {
        try {
            const response = await fetch('/api/stats');
            const data = await response.json();
            if (data.success) {
                elements.stats.innerHTML = `
                    ${data.data.activeGames} active games · ${data.data.totalPlayers} players online
                `;
            }
        } catch (e) {
            // Silently fail
        }
    }

    // ============================================
    // Initialize
    // ============================================
    function init() {
        setupEventListeners();
        handleRouting();
        loadStats();

        // Periodically update stats
        setInterval(loadStats, 30000);
    }

    // Start the app
    document.addEventListener('DOMContentLoaded', init);
})();

