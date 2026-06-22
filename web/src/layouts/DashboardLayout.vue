<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ref } from 'vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const isSidebarCollapsed = ref(false)

const navItems = [
  { path: '/overview', label: 'Overview', icon: '◈' },
  { path: '/origins', label: 'Origins', icon: '◉' },
  { path: '/rules', label: 'Rules', icon: '◆' },
  { path: '/explorer', label: 'Storage Explorer', icon: '◫' },
  { path: '/unknown-origins', label: 'Unknown Origins', icon: '◇' },
]

function handleLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="dashboard">
    <!-- Sidebar -->
    <aside class="sidebar" :class="{ 'sidebar--collapsed': isSidebarCollapsed }">
      <div class="sidebar__header">
        <div class="sidebar__brand">
          <span class="sidebar__logo">◈</span>
          <span v-show="!isSidebarCollapsed" class="sidebar__title">LumbungFS</span>
        </div>
        <button class="sidebar__toggle" @click="isSidebarCollapsed = !isSidebarCollapsed">
          {{ isSidebarCollapsed ? '→' : '←' }}
        </button>
      </div>

      <nav class="sidebar__nav">
        <RouterLink
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="sidebar__link"
          :class="{ 'sidebar__link--active': route.path === item.path }"
        >
          <span class="sidebar__link-icon">{{ item.icon }}</span>
          <span v-show="!isSidebarCollapsed" class="sidebar__link-label">{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="sidebar__footer">
        <div v-show="!isSidebarCollapsed" class="sidebar__user">
          <span class="sidebar__user-avatar">{{ auth.username.charAt(0).toUpperCase() }}</span>
          <span class="sidebar__user-name">{{ auth.username }}</span>
        </div>
        <button class="sidebar__logout" @click="handleLogout" title="Logout">
          ⏻
        </button>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="main">
      <header class="topbar">
        <div class="topbar__breadcrumb">
          <span class="topbar__section-icon">{{ navItems.find(n => n.path === route.path)?.icon || '◈' }}</span>
          <span class="topbar__section-label">{{ navItems.find(n => n.path === route.path)?.label || 'Dashboard' }}</span>
        </div>
        <div class="topbar__actions">
          <span class="topbar__status">
            <span class="topbar__status-dot"></span>
            System Online
          </span>
        </div>
      </header>
      <div class="content">
        <RouterView />
      </div>
    </main>
  </div>
</template>

<style scoped>
.dashboard {
  display: flex;
  height: 100vh;
  background: var(--color-sage-paper);
  overflow: hidden;
}

/* ───── Sidebar ───── */
.sidebar {
  display: flex;
  flex-direction: column;
  width: 240px;
  background: var(--color-forest-ink);
  color: var(--color-bone-white);
  transition: width 0.25s ease;
  flex-shrink: 0;
  z-index: 10;
}
.sidebar--collapsed {
  width: 64px;
}
.sidebar__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-20) var(--spacing-16);
  border-bottom: 0.5px solid rgba(202, 211, 210, 0.12);
}
.sidebar__brand {
  display: flex;
  align-items: center;
  gap: var(--spacing-8);
  min-width: 0;
}
.sidebar__logo {
  font-size: 20px;
  color: var(--color-moss);
  flex-shrink: 0;
}
.sidebar__title {
  font-family: var(--font-cinetype);
  font-size: 13px;
  font-weight: 500;
  letter-spacing: 0.24em;
  text-transform: uppercase;
  white-space: nowrap;
}
.sidebar__toggle {
  background: none;
  border: none;
  color: var(--color-lichen);
  cursor: pointer;
  font-size: 14px;
  padding: 2px 6px;
  border-radius: var(--radius-md);
  flex-shrink: 0;
}
.sidebar__toggle:hover {
  background: rgba(255, 255, 255, 0.06);
}

/* ───── Nav ───── */
.sidebar__nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--spacing-12) var(--spacing-8);
  overflow-y: auto;
}
.sidebar__link {
  display: flex;
  align-items: center;
  gap: var(--spacing-12);
  padding: 10px 12px;
  border-radius: var(--radius-xl);
  font-family: var(--font-muoto);
  font-size: 14px;
  font-weight: 400;
  color: var(--color-lichen);
  text-decoration: none;
  transition: background 0.15s ease, color 0.15s ease;
}
.sidebar__link:hover {
  background: rgba(255, 255, 255, 0.06);
  color: var(--color-bone-white);
}
.sidebar__link--active {
  background: rgba(133, 192, 147, 0.12);
  color: var(--color-moss);
}
.sidebar__link-icon {
  font-size: 16px;
  width: 20px;
  text-align: center;
  flex-shrink: 0;
}
.sidebar__link-label {
  white-space: nowrap;
}

/* ───── Footer ───── */
.sidebar__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-16);
  border-top: 0.5px solid rgba(202, 211, 210, 0.12);
}
.sidebar__user {
  display: flex;
  align-items: center;
  gap: var(--spacing-8);
  min-width: 0;
}
.sidebar__user-avatar {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--color-moss);
  color: var(--color-forest-ink);
  font-family: var(--font-denim);
  font-weight: 600;
  font-size: 13px;
  flex-shrink: 0;
}
.sidebar__user-name {
  font-family: var(--font-muoto);
  font-size: 13px;
  color: var(--color-lichen);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.sidebar__logout {
  background: none;
  border: none;
  color: var(--color-lichen);
  cursor: pointer;
  font-size: 16px;
  padding: 4px;
  border-radius: var(--radius-md);
  flex-shrink: 0;
}
.sidebar__logout:hover {
  color: #c44d4d;
}

/* ───── Main ───── */
.main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-16) var(--spacing-32);
  background: var(--color-bone-white);
  border-bottom: 0.5px solid var(--color-lichen);
  flex-shrink: 0;
}
.topbar__breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--spacing-8);
}
.topbar__section-icon {
  font-size: 16px;
  color: var(--color-moss);
}
.topbar__section-label {
  font-family: var(--font-denim);
  font-size: 16px;
  font-weight: 500;
  color: var(--color-forest-ink);
}
.topbar__actions {
  display: flex;
  align-items: center;
  gap: var(--spacing-16);
}
.topbar__status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-muoto);
  font-size: 12px;
  color: var(--color-slate-smoke);
}
.topbar__status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-deep-fern);
  animation: pulse-dot 2s infinite;
}

@keyframes pulse-dot {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.content {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-32);
}
</style>
