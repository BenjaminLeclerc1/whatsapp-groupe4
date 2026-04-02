import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Counter, Rate, Trend, Gauge } from 'k6/metrics';

// ─── Métriques custom ─────────────────────────────────────────────────────────
const connexionsActives  = new Gauge('connexions_actives');
const connexionsReussies = new Counter('connexions_reussies');
const connexionsEchouees = new Counter('connexions_echouees');
const loginSuccessRate   = new Rate('login_success_rate');
const tempsConnexion     = new Trend('temps_connexion_ms', true);

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// ─── 100 000 connexions simultanées ──────────────────────────────────────────
// Sur Kubernetes : 100 pods × 1 000 VUs = 100 000 connexions
// Sur Minikube local : réduis à vus: 1000 pour tester sans k8s
export const options = {
  scenarios: {

    // ── TEST 1 : 100k connexions simultanées exactes ─────────────────────────
    cent_mille_connexions: {
      executor: 'constant-vus',
      vus: 100000,               // ← 100 000 connexions simultanées
      duration: '3m',
      tags: { test: '100k_simultane' },
    },

    // ── TEST 2 : montée progressive jusqu'à 100k ─────────────────────────────
    montee_progressive: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '1m',  target: 10000  },  // 10k connexions
        { duration: '1m',  target: 25000  },  // 25k connexions
        { duration: '1m',  target: 50000  },  // 50k connexions
        { duration: '1m',  target: 75000  },  // 75k connexions
        { duration: '1m',  target: 100000 },  // 100k connexions ← objectif
        { duration: '1m',  target: 100000 },  // maintien 1 minute
        { duration: '30s', target: 0      },  // descente
      ],
      startTime: '4m',
      tags: { test: 'montee_progressive' },
    },

    // ── TEST 3 : spike brutal à 100k connexions ──────────────────────────────
    spike_100k: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '15s', target: 100000 },  // 100k d'un coup
        { duration: '2m',  target: 100000 },  // maintien 2 minutes
        { duration: '15s', target: 0      },  // arrêt brutal
      ],
      startTime: '12m',
      tags: { test: 'spike' },
    },
  },

  thresholds: {
    http_req_failed:                          ['rate<0.01'],
    http_req_duration:                        ['p(95)<2000'],
    'http_req_duration{test:100k_simultane}': ['p(99)<5000'],
    login_success_rate:                       ['rate>0.95'],
    temps_connexion_ms:                       ['p(95)<1000'],
  },
};

export default function () {
  connexionsActives.add(1);

  const email    = `user${__VU}_${__ITER}@test.dev`;
  const password = 'Test1234!';
  const username = `u${__VU}_${__ITER}`;

  // ── 1. Connexion ────────────────────────────────────────────────────────────
  let token = null;

  group('1. Connexion', () => {
    const start = Date.now();

    const res = http.post(
      `${BASE_URL}/api/v1/auth/register`,
      JSON.stringify({ username, email, password }),
      { headers: { 'Content-Type': 'application/json' } }
    );

    const ok = check(res, {
      'register: 200/201': (r) => r.status === 200 || r.status === 201,
    });

    if (!ok) {
      connexionsEchouees.add(1);
      loginSuccessRate.add(false);
      return;
    }

    try { token = JSON.parse(res.body).access_token; } catch {}
    tempsConnexion.add(Date.now() - start);
    connexionsReussies.add(1);
    loginSuccessRate.add(true);
  });

  if (!token) { connexionsActives.add(-1); return; }

  // ── 2. Session active (30 secondes d'activité) ──────────────────────────────
  const sessionDuration = 30;
  const startSession    = Date.now();

  while ((Date.now() - startSession) / 1000 < sessionDuration) {
    group('2. Activité session', () => {
      // Vérifier la session
      const meRes = http.get(
        `${BASE_URL}/api/v1/auth/me`,
        { headers: { 'Authorization': `Bearer ${token}` } }
      );
      check(meRes, { 'session active': (r) => r.status === 200 });

      sleep(0.5);

      // Récupérer les chats
      const chatsRes = http.get(
        `${BASE_URL}/api/v1/chats`,
        { headers: { 'Authorization': `Bearer ${token}` } }
      );
      check(chatsRes, { 'chats: 200': (r) => r.status === 200 });

      sleep(0.5);

      // Health check
      check(
        http.get(`${BASE_URL}/api/v1/health`),
        { 'health: 200': (r) => r.status === 200 }
      );
    });

    sleep(2);
  }

  // ── 3. Déconnexion ─────────────────────────────────────────────────────────
  group('3. Déconnexion', () => {
    http.post(
      `${BASE_URL}/api/v1/auth/logout`,
      JSON.stringify({}),
      { headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' } }
    );
  });

  connexionsActives.add(-1);
}

export function handleSummary(data) {
  const reussies  = data.metrics.connexions_reussies?.values?.count || 0;
  const echouees  = data.metrics.connexions_echouees?.values?.count || 0;
  const p95conn   = data.metrics.temps_connexion_ms?.values['p(95)']?.toFixed(0) || '?';
  const p95req    = data.metrics.http_req_duration?.values['p(95)']?.toFixed(0) || '?';
  const errRate   = ((data.metrics.http_req_failed?.values?.rate || 0) * 100).toFixed(2);
  const loginRate = ((data.metrics.login_success_rate?.values?.rate || 0) * 100).toFixed(2);

  const verdict = errRate < 1 && loginRate > 95
    ? '✅ SUCCÈS — L\'app tient 100 000 connexions simultanées'
    : '❌ ÉCHEC  — L\'app ne tient pas 100k connexions';

  console.log('\n╔══════════════════════════════════════════════════════╗');
  console.log('║    RÉSULTAT — 100 000 CONNEXIONS SIMULTANÉES         ║');
  console.log('╠══════════════════════════════════════════════════════╣');
  console.log(`║  Connexions réussies  : ${String(reussies).padEnd(26)} ║`);
  console.log(`║  Connexions échouées  : ${String(echouees).padEnd(26)} ║`);
  console.log(`║  Taux de login        : ${String(loginRate + '%').padEnd(26)} ║`);
  console.log(`║  Taux d'erreur HTTP   : ${String(errRate + '%').padEnd(26)} ║`);
  console.log(`║  P95 temps connexion  : ${String(p95conn + 'ms').padEnd(26)} ║`);
  console.log(`║  P95 durée requête    : ${String(p95req + 'ms').padEnd(26)} ║`);
  console.log('╠══════════════════════════════════════════════════════╣');
  console.log(`║  ${verdict.padEnd(53)}║`);
  console.log('╚══════════════════════════════════════════════════════╝\n');

  return {
    'k6/results/concurrent-100k-summary.json': JSON.stringify(data, null, 2),
    stdout: `\n${verdict}\n`,
  };
}
