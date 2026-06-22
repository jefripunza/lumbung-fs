package templates

const ExpiredTokenHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Invalid or Expired Link - LumbungFS</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=Outfit:wght@500;600&display=swap" rel="stylesheet">
    <style>
        :root {
            --color-deep-forest: #0c322c;
            --color-bone-white: #faf9f5;
            --color-forest-ink: #0e2a22;
            --color-slate-smoke: #62756f;
            --color-lichen: #cad3d2;
            --color-moss: #5c8e75;
            --color-deep-fern: #2d4f45;
            --radius-xl: 16px;
            --radius-md: 8px;
        }
        
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Inter', -apple-system, sans-serif;
            background: var(--color-deep-forest);
            color: var(--color-forest-ink);
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            padding: 20px;
        }

        .error-card {
            background: var(--color-bone-white);
            border-radius: var(--radius-xl);
            box-shadow: 0 10px 40px rgba(0,0,0,0.3);
            width: 100%;
            max-width: 480px;
            padding: 48px 32px;
            text-align: center;
            animation: slideUp 0.3s ease-out;
        }

        @keyframes slideUp {
            from { opacity: 0; transform: translateY(12px); }
            to { opacity: 1; transform: translateY(0); }
        }

        .error-icon {
            font-size: 56px;
            margin-bottom: 20px;
            display: inline-block;
        }

        .title {
            font-family: 'Outfit', sans-serif;
            font-size: 24px;
            font-weight: 600;
            color: #9b1c1c;
            margin-bottom: 12px;
        }

        .desc {
            font-size: 14px;
            color: var(--color-slate-smoke);
            line-height: 1.6;
            margin-bottom: 32px;
        }

        .footer {
            border-top: 0.5px solid var(--color-lichen);
            padding-top: 20px;
            font-size: 12px;
            color: var(--color-slate-smoke);
        }
    </style>
</head>
<body>
    <div class="error-card">
        <span class="error-icon">⚠️</span>
        <h1 class="title">Upload Link Expired</h1>
        <p class="desc">The upload link you followed is invalid, expired, or has already been used. Presigned upload links expire automatically after 1 minute for security.</p>
        <div class="footer">
            Powered by LumbungFS
        </div>
    </div>
</body>
</html>`
