<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import MossButton from '@/components/MossButton.vue'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')

async function handleLogin() {
  const ok = await auth.login(username.value, password.value)
  if (ok) {
    router.push('/overview')
  }
}
</script>

<template>
  <div class="login-page">
    <!-- Dot grid background -->
    <div class="login-page__grid" aria-hidden="true"></div>

    <div class="login-card">
      <!-- Brand -->
      <div class="login-card__brand">
        <span class="login-card__logo">◈</span>
        <h1 class="login-card__title">LUMBUNGFS</h1>
        <p class="login-card__subtitle">Infrastructure Control Center</p>
      </div>

      <!-- Form -->
      <form class="login-card__form" @submit.prevent="handleLogin">
        <div class="field">
          <label class="field__label" for="login-username">USERNAME</label>
          <input
            id="login-username"
            v-model="username"
            type="text"
            class="field__input"
            placeholder="admin"
            autocomplete="username"
            required
          />
        </div>

        <div class="field">
          <label class="field__label" for="login-password">PASSWORD</label>
          <input
            id="login-password"
            v-model="password"
            type="password"
            class="field__input"
            placeholder="••••••••"
            autocomplete="current-password"
            required
          />
        </div>

        <p v-if="auth.loginError" class="login-card__error">{{ auth.loginError }}</p>

        <MossButton type="submit" :disabled="auth.isLoading" class="login-card__submit">
          {{ auth.isLoading ? 'Authenticating…' : 'Authenticate' }}
        </MossButton>
      </form>

      <!-- Footer -->
      <p class="login-card__footer">Secure file storage infrastructure</p>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: var(--color-forest-ink);
  position: relative;
  overflow: hidden;
}

/* Dot grid pattern */
.login-page__grid {
  position: absolute;
  inset: 0;
  background-image: radial-gradient(circle, rgba(133, 192, 147, 0.08) 1px, transparent 1px);
  background-size: 24px 24px;
  pointer-events: none;
}

.login-card {
  position: relative;
  background: var(--color-bone-white);
  border-radius: var(--radius-3xl);
  padding: var(--spacing-48);
  width: 420px;
  max-width: 92vw;
  box-shadow:
    0 0 60px rgba(0, 0, 0, 0.15),
    var(--shadow-subtle);
  animation: card-enter 0.4s ease-out;
}

/* ───── Brand ───── */
.login-card__brand {
  text-align: center;
  margin-bottom: var(--spacing-40);
}
.login-card__logo {
  display: inline-block;
  font-size: 36px;
  color: var(--color-moss);
  margin-bottom: var(--spacing-12);
}
.login-card__title {
  font-family: var(--font-cinetype);
  font-size: 18px;
  font-weight: 500;
  letter-spacing: 0.32em;
  color: var(--color-forest-ink);
  margin: 0;
}
.login-card__subtitle {
  font-family: var(--font-muoto);
  font-size: 12px;
  color: var(--color-slate-smoke);
  margin: 6px 0 0;
  letter-spacing: 0.02em;
}

/* ───── Form ───── */
.login-card__form {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-20);
}
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field__label {
  font-family: var(--font-cinetype);
  font-size: 11px;
  font-weight: 400;
  letter-spacing: 0.20em;
  color: var(--color-slate-smoke);
  text-transform: uppercase;
}
.field__input {
  background: var(--color-sage-paper);
  border: 0.5px solid var(--color-lichen);
  border-radius: var(--radius-xl);
  padding: 12px 16px;
  font-family: var(--font-denim);
  font-size: 14px;
  color: var(--color-forest-ink);
  outline: none;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}
.field__input::placeholder {
  color: var(--color-lichen);
}
.field__input:focus {
  border-color: var(--color-moss);
  box-shadow: 0 0 0 3px rgba(133, 192, 147, 0.15);
}

.login-card__error {
  font-family: var(--font-muoto);
  font-size: 13px;
  color: #c44d4d;
  margin: 0;
  padding: 8px 12px;
  background: rgba(196, 77, 77, 0.06);
  border-radius: var(--radius-md);
  border: 0.5px solid rgba(196, 77, 77, 0.2);
}

.login-card__submit {
  width: 100%;
  margin-top: var(--spacing-8);
  padding: 14px;
  font-size: 15px;
}

/* ───── Footer ───── */
.login-card__footer {
  text-align: center;
  font-family: var(--font-muoto);
  font-size: 11px;
  color: var(--color-lichen);
  margin-top: var(--spacing-24);
}

@keyframes card-enter {
  from {
    transform: translateY(16px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}
</style>
