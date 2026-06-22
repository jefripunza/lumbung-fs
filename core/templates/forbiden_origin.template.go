package templates

const ForbiddenOriginTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Forbidden - Unknown Origin | LumbungFS</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@400;600;800&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-color: #0b0f19;
            --card-bg: rgba(17, 24, 39, 0.7);
            --border-color: rgba(255, 255, 255, 0.08);
            --text-primary: #f3f4f6;
            --text-secondary: #9ca3af;
            --accent-glow: radial-gradient(circle, rgba(99, 102, 241, 0.15) 0%, rgba(168, 85, 247, 0.05) 50%, transparent 100%);
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Outfit', sans-serif;
            background-color: var(--bg-color);
            color: var(--text-primary);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            overflow: hidden;
            position: relative;
        }

        .glow-1 {
            position: absolute;
            width: 600px;
            height: 600px;
            top: -200px;
            left: -200px;
            background: radial-gradient(circle, rgba(99, 102, 241, 0.12) 0%, transparent 70%);
            filter: blur(80px);
            z-index: 0;
            pointer-events: none;
        }

        .glow-2 {
            position: absolute;
            width: 600px;
            height: 600px;
            bottom: -200px;
            right: -200px;
            background: radial-gradient(circle, rgba(168, 85, 247, 0.12) 0%, transparent 70%);
            filter: blur(80px);
            z-index: 0;
            pointer-events: none;
        }

        .container {
            z-index: 10;
            width: 100%;
            max-width: 520px;
            padding: 20px;
            text-align: center;
        }

        .card {
            background: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: 24px;
            padding: 40px 32px;
            backdrop-filter: blur(20px);
            -webkit-backdrop-filter: blur(20px);
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3), 
                        inset 0 1px 1px rgba(255, 255, 255, 0.1);
            animation: fadeIn 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards;
        }

        .icon-container {
            width: 80px;
            height: 80px;
            background: rgba(239, 68, 68, 0.1);
            border: 1px solid rgba(239, 68, 68, 0.2);
            border-radius: 20px;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0 auto 24px;
            color: #ef4444;
            box-shadow: 0 8px 24px rgba(239, 68, 68, 0.1);
            animation: pulse 2s infinite;
        }

        .title {
            font-size: 28px;
            font-weight: 800;
            margin-bottom: 12px;
            background: linear-gradient(135deg, #f43f5e 0%, #a855f7 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            letter-spacing: -0.5px;
        }

        .description {
            font-size: 16px;
            line-height: 1.6;
            color: var(--text-secondary);
            margin-bottom: 28px;
        }

        .domain-badge {
            display: inline-block;
            background: rgba(255, 255, 255, 0.05);
            border: 1px solid rgba(255, 255, 255, 0.08);
            border-radius: 12px;
            padding: 8px 16px;
            font-family: monospace;
            font-size: 15px;
            color: #a855f7;
            margin-top: 8px;
            word-break: break-all;
        }

        .divider {
            height: 1px;
            background: linear-gradient(90deg, transparent, var(--border-color), transparent);
            margin: 24px 0;
        }

        .steps {
            text-align: left;
            margin-bottom: 32px;
        }

        .step-item {
            display: flex;
            align-items: flex-start;
            margin-bottom: 12px;
            font-size: 14px;
            color: var(--text-secondary);
        }

        .step-number {
            background: rgba(168, 85, 247, 0.15);
            color: #c084fc;
            width: 24px;
            height: 24px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 600;
            font-size: 12px;
            margin-right: 12px;
            flex-shrink: 0;
        }

        .step-text {
            line-height: 1.5;
            padding-top: 2px;
        }

        .btn {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            width: 100%;
            padding: 14px 28px;
            background: linear-gradient(135deg, #6366f1 0%, #a855f7 100%);
            border: none;
            border-radius: 12px;
            color: white;
            font-weight: 600;
            font-size: 16px;
            cursor: pointer;
            text-decoration: none;
            transition: all 0.3s ease;
            box-shadow: 0 4px 20px rgba(99, 102, 241, 0.25);
        }

        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 28px rgba(99, 102, 241, 0.4);
            opacity: 0.95;
        }

        .btn:active {
            transform: translateY(0);
        }

        .logo {
            font-weight: 800;
            font-size: 18px;
            color: var(--text-primary);
            letter-spacing: -0.5px;
            margin-top: 20px;
            display: inline-flex;
            align-items: center;
            gap: 8px;
            opacity: 0.7;
        }

        .logo-dot {
            width: 8px;
            height: 8px;
            background: #a855f7;
            border-radius: 50%;
        }

        @keyframes fadeIn {
            from {
                opacity: 0;
                transform: translateY(20px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        @keyframes pulse {
            0% {
                box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.4);
            }
            70% {
                box-shadow: 0 0 0 12px rgba(239, 68, 68, 0);
            }
            100% {
                box-shadow: 0 0 0 0 rgba(239, 68, 68, 0);
            }
        }
    </style>
</head>
<body>
    <div class="glow-1"></div>
    <div class="glow-2"></div>
    <div class="container">
        <div class="card">
            <div class="icon-container">
                <svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
            </div>
            <h1 class="title">Unknown Origin Domain</h1>
            <p class="description">
                Access to serve files from this origin is blocked because it has not been registered in the system.
                <br>
                <span class="domain-badge">{{.Domain}}</span>
            </p>
            
            <div class="divider"></div>
            
            <div class="steps">
                <div class="step-item">
                    <span class="step-number">1</span>
                    <span class="step-text">Open your LumbungFS Admin Dashboard</span>
                </div>
                <div class="step-item">
                    <span class="step-number">2</span>
                    <span class="step-text">Go to the <strong>Origins</strong> tab</span>
                </div>
                <div class="step-item">
                    <span class="step-number">3</span>
                    <span class="step-text">Add and register <strong style="color: #c084fc;">{{.Domain}}</strong></span>
                </div>
            </div>

            <a href="javascript:void(0);" onclick="goToDashboard()" class="btn">Go to Dashboard</a>
        </div>
        
        <div class="logo">
            <div class="logo-dot"></div>
            LumbungFS
        </div>
    </div>

    <script>
        function goToDashboard() {
            const currentPort = window.location.port;
            if (currentPort === '8080') {
                window.location.href = window.location.protocol + '//' + window.location.hostname + ':5173';
            } else {
                window.location.href = '/';
            }
        }
    </script>
</body>
</html>`
