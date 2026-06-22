package templates

const UploaderHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Upload File - LumbungFS</title>
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

        .upload-card {
            background: var(--color-bone-white);
            border-radius: var(--radius-xl);
            box-shadow: 0 10px 40px rgba(0,0,0,0.3);
            width: 100%;
            max-width: 480px;
            padding: 40px 32px;
            text-align: center;
            animation: slideUp 0.3s ease-out;
        }

        @keyframes slideUp {
            from { opacity: 0; transform: translateY(12px); }
            to { opacity: 1; transform: translateY(0); }
        }

        .header {
            margin-bottom: 28px;
        }

        .title {
            font-family: 'Outfit', sans-serif;
            font-size: 24px;
            font-weight: 600;
            color: var(--color-deep-fern);
            margin-bottom: 8px;
        }

        .subtitle {
            font-size: 14px;
            color: var(--color-slate-smoke);
            line-height: 1.5;
        }

        .origin-badge {
            display: inline-block;
            background: rgba(92, 142, 117, 0.12);
            color: var(--color-deep-fern);
            padding: 4px 8px;
            border-radius: var(--radius-md);
            font-size: 12px;
            font-weight: 500;
            margin-top: 10px;
            border: 0.5px solid var(--color-lichen);
        }

        .dropzone {
            border: 2px dashed var(--color-lichen);
            background: #fdfdfb;
            border-radius: var(--radius-xl);
            padding: 40px 20px;
            cursor: pointer;
            transition: all 0.2s ease;
            position: relative;
            margin-bottom: 24px;
        }

        .dropzone:hover, .dropzone.dragover {
            border-color: var(--color-moss);
            background: #f7f9f7;
        }

        .dropzone-icon {
            font-size: 40px;
            margin-bottom: 16px;
            display: inline-block;
        }

        .dropzone-text {
            font-size: 14px;
            font-weight: 500;
            color: var(--color-forest-ink);
            margin-bottom: 6px;
        }

        .dropzone-subtext {
            font-size: 12px;
            color: var(--color-slate-smoke);
        }

        .file-input {
            display: none;
        }

        .progress-container {
            display: none;
            margin-bottom: 24px;
            text-align: left;
        }

        .progress-label {
            display: flex;
            justify-content: space-between;
            font-size: 12px;
            font-weight: 500;
            color: var(--color-forest-ink);
            margin-bottom: 6px;
        }

        .progress-track {
            height: 8px;
            background: var(--color-lichen);
            border-radius: 4px;
            overflow: hidden;
        }

        .progress-bar {
            height: 100%;
            width: 0%;
            background: linear-gradient(90deg, var(--color-moss), var(--color-deep-fern));
            transition: width 0.1s ease;
            border-radius: 4px;
        }

        .error-message {
            display: none;
            background: #fdf2f2;
            border: 0.5px solid #f8b4b4;
            color: #9b1c1c;
            padding: 12px;
            border-radius: var(--radius-md);
            font-size: 13px;
            margin-bottom: 24px;
            text-align: left;
            line-height: 1.4;
        }

        .success-container {
            display: none;
            animation: fadeIn 0.2s ease-out;
        }

        @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
        }

        .success-icon {
            font-size: 48px;
            margin-bottom: 16px;
            display: inline-block;
        }

        .success-title {
            font-family: 'Outfit', sans-serif;
            font-size: 20px;
            font-weight: 600;
            color: #1e3a2f;
            margin-bottom: 8px;
        }

        .success-desc {
            font-size: 13.5px;
            color: var(--color-slate-smoke);
            margin-bottom: 20px;
        }

        .url-box {
            display: flex;
            background: #f3f2ee;
            border: 0.5px solid var(--color-lichen);
            border-radius: var(--radius-md);
            padding: 10px 12px;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 24px;
            text-align: left;
        }

        .url-text {
            font-size: 13px;
            color: var(--color-forest-ink);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            margin-right: 12px;
            flex: 1;
        }

        .copy-btn {
            background: var(--color-moss);
            border: none;
            border-radius: 6px;
            color: white;
            padding: 6px 12px;
            font-size: 12px;
            font-weight: 500;
            cursor: pointer;
            transition: background 0.15s ease;
            flex-shrink: 0;
        }

        .copy-btn:hover {
            background: var(--color-deep-fern);
        }
    </style>
</head>
<body>
    <div class="upload-card">
        <div id="main-flow">
            <div class="header">
                <h1 class="title">Upload File</h1>
                <p class="subtitle">Upload a file directly to the storage bucket using your temporary 1-minute presigned token</p>
                <div class="origin-badge">🌐 {{.Domain}}{{.PathSuffix}}</div>
            </div>

            <div id="error-box" class="error-message"></div>

            <div class="dropzone" id="dropzone">
                <span class="dropzone-icon">📥</span>
                <p class="dropzone-text">Drag & drop your file here</p>
                <p class="dropzone-subtext">or click to browse from device</p>
                <input type="file" id="file-input" class="file-input">
            </div>

            <div class="progress-container" id="progress-container">
                <div class="progress-label">
                    <span id="progress-filename">Uploading...</span>
                    <span id="progress-percent">0%</span>
                </div>
                <div class="progress-track">
                    <div class="progress-bar" id="progress-bar"></div>
                </div>
            </div>
        </div>

        <div id="success-flow" class="success-container">
            <span class="success-icon">🎉</span>
            <h2 class="success-title">Upload Successful!</h2>
            <p class="success-desc">Your file has been uploaded and stored securely. You can copy the public access URL below:</p>
            
            <div class="url-box">
                <span class="url-text" id="url-text">https://example.com/file/photo.jpg</span>
                <button type="button" class="copy-btn" id="copy-btn">Copy Link</button>
            </div>
        </div>
    </div>

    <script>
        const dropzone = document.getElementById('dropzone');
        const fileInput = document.getElementById('file-input');
        const progressContainer = document.getElementById('progress-container');
        const progressBar = document.getElementById('progress-bar');
        const progressPercent = document.getElementById('progress-percent');
        const progressFilename = document.getElementById('progress-filename');
        const errorBox = document.getElementById('error-box');
        const mainFlow = document.getElementById('main-flow');
        const successFlow = document.getElementById('success-flow');
        const urlText = document.getElementById('url-text');
        const copyBtn = document.getElementById('copy-btn');

        const params = new URLSearchParams(window.location.search);
        const token = params.get('token');

        if (!token) {
            showError("Access Denied: Missing presigned upload token.");
            dropzone.style.pointerEvents = 'none';
            dropzone.style.opacity = '0.5';
        }

        ['dragenter', 'dragover'].forEach(eventName => {
            dropzone.addEventListener(eventName, e => {
                e.preventDefault();
                dropzone.classList.add('dragover');
            }, false);
        });

        ['dragleave', 'drop'].forEach(eventName => {
            dropzone.addEventListener(eventName, e => {
                e.preventDefault();
                dropzone.classList.remove('dragover');
            }, false);
        });

        dropzone.addEventListener('drop', e => {
            const dt = e.dataTransfer;
            const files = dt.files;
            if (files.length) {
                handleFile(files[0]);
            }
        });

        dropzone.addEventListener('click', () => {
            fileInput.click();
        });

        fileInput.addEventListener('change', () => {
            if (fileInput.files.length) {
                handleFile(fileInput.files[0]);
            }
        });

        function showError(msg) {
            errorBox.innerText = msg;
            errorBox.style.display = 'block';
        }

        function handleFile(file) {
            if (!token) return;
            
            errorBox.style.display = 'none';
            dropzone.style.display = 'none';
            progressContainer.style.display = 'block';
            progressFilename.innerText = file.name;

            const formData = new FormData();
            formData.append('file', file);

            const xhr = new XMLHttpRequest();
            xhr.open('POST', "/upload?token=" + encodeURIComponent(token), true);

            xhr.upload.addEventListener('progress', e => {
                if (e.lengthComputable) {
                    const percent = Math.round((e.loaded * 100) / e.total);
                    progressBar.style.width = percent + '%';
                    progressPercent.innerText = percent + '%';
                }
            });

            xhr.onload = () => {
                if (xhr.status === 201) {
                    try {
                        const res = JSON.parse(xhr.responseText);
                        showSuccess(res.url);
                    } catch (err) {
                        showError("Failed to parse server response.");
                        resetFlow();
                    }
                } else {
                    let errMsg = "Upload failed.";
                    try {
                        const res = JSON.parse(xhr.responseText);
                        if (res.error) errMsg = res.error;
                    } catch (e) {}
                    showError(errMsg);
                    resetFlow();
                }
            };

            xhr.onerror = () => {
                showError("Network error occurred.");
                resetFlow();
            };

            xhr.send(formData);
        }

        function resetFlow() {
            progressContainer.style.display = 'none';
            dropzone.style.display = 'block';
        }

        function showSuccess(url) {
            mainFlow.style.display = 'none';
            successFlow.style.display = 'block';
            urlText.innerText = url;
        }

        copyBtn.addEventListener('click', async () => {
            try {
                await navigator.clipboard.writeText(urlText.innerText);
                copyBtn.innerText = "Copied!";
                setTimeout(() => {
                    copyBtn.innerText = "Copy Link";
                }, 2000);
            } catch (err) {
                console.error("Failed to copy", err);
            }
        });
    </script>
</body>
</html>`
