<div align="center">
  <h1>🚀 DARFIN</h1>
  <p><b>Dynamic Accelerator of Resource & File Integration Network</b></p>
  <p>An open-source, lightning-fast, and modern desktop download manager built with Golang and Wails.</p>
</div>

---

## ✨ Features

- ⚡ **Multi-threaded Downloads**: Accelerate your download speeds by splitting files into multiple concurrent parts.
- ⏸️ **Resume Capability**: Seamlessly pause and resume broken or stopped downloads without losing progress.
- 🌐 **Browser Integration**: Catch downloads directly from your browser using the provided Chrome and Firefox extensions.
- 🗓️ **Queue & Scheduling**: Organize your downloads in a queue and schedule them to start at your convenience.
- 🎨 **Modern UI**: Beautiful, responsive, and dark-themed user interface built with React & TypeScript.
- 🪶 **Lightweight**: Uses Wails to provide a desktop app native experience without the memory overhead of Electron.

## 🛠️ Tech Stack

- **Core Engine**: [Golang](https://go.dev/)
- **Desktop Framework**: [Wails v2](https://wails.io/)
- **Frontend**: [React 18](https://reactjs.org/), [TypeScript](https://www.typescriptlang.org/), [Vite](https://vitejs.dev/)
- **Icons**: [Lucide React](https://lucide.dev/)

## 🚀 Getting Started

### ⬇️ Download Pre-built Release (Recommended for Users)
The easiest way to install DARFIN is to download the pre-compiled executable directly.
You do **not** need to install Go, Node.js, or any other tools.
Just find the latest version on our [Releases page](https://github.com/abdlhli/DARFIN/releases) and run it!

---

### 🛠️ Build from Source (For Developers)

#### Prerequisites

Before you begin, ensure you have met the following requirements:
- **Go 1.20+** installed.
- **Node.js 16+** and **npm** installed.
- **Wails CLI** installed:
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

#### Installation & Build

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/DARFIN.git
   cd DARFIN
   ```

2. **Run in Development Mode**
   This will start the Wails development server with live reload for both frontend and backend.
   ```bash
   wails dev
   ```

3. **Build the Application**
   To build a production-ready application for your current operating system:
   ```bash
   wails build
   ```
   The compiled executable will be generated in the `build/bin/` folder.

## 🔌 Browser Extensions

DARFIN seamlessly integrates with your favorite browsers to automatically intercept file downloads. 
To install the extensions in developer mode:

- **Chrome / Edge / Brave**: 
  1. Go to `chrome://extensions/`
  2. Enable "Developer mode"
  3. Click "Load unpacked" and select the `browser-extension` folder in this repository.
  
- **Firefox**: 
  1. Go to `about:debugging#/runtime/this-firefox`
  2. Click "Load Temporary Add-on..."
  3. Select any file inside the `firefox-extension` folder.

## 🤝 Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.

---
<div align="center">
  <p>Built with ❤️ by the open source community.</p>
</div>
