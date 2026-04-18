<template>
  <div class="home-layout">

    <!-- Query list -->
    <div class="query-list-panel">
      <div class="panel-title">Queries</div>
      <div v-if="loadingQueries" class="empty-hint">Loading…</div>
      <div v-else-if="!queries.length" class="empty-hint">No queries available.</div>
      <div v-else>
        <div
          v-for="q in queries"
          :key="q.id"
          class="query-item"
          :class="{ active: selectedQuery?.id === q.id }"
          @click="selectQuery(q)"
        >
          <div class="query-item-name">{{ q.name }}</div>
          <div v-if="q.description" class="query-item-desc">{{ q.description }}</div>
        </div>
      </div>
    </div>

    <!-- Run panel -->
    <div class="run-panel">
      <div v-if="!selectedQuery" class="centered-hint">
        <v-icon size="40" color="grey-lighten-1">mdi-database-search</v-icon>
        <div class="mt-2">Select a query to get started</div>
      </div>

      <template v-else>
        <!-- Query header -->
        <div class="run-header">
          <div>
            <div class="run-title">{{ selectedQuery.name }}</div>
            <div v-if="selectedQuery.description" class="run-desc">{{ selectedQuery.description }}</div>
          </div>
          <v-btn color="primary" :loading="running" @click="run" prepend-icon="mdi-play">Run</v-btn>
        </div>

        <!-- Parameters -->
        <div v-if="paramDefs.length" class="params-area">
          <div class="section-label mb-3">Parameters</div>
          <v-row dense>
            <v-col v-for="p in paramDefs" :key="p.name" cols="12" sm="6" md="4">
              <v-select
                v-if="p.type === 'bool'"
                v-model="paramValues[p.name]"
                :label="p.name + (p.required ? ' *' : '')"
                :items="[{title:'—',value:''},{title:'true',value:'true'},{title:'false',value:'false'}]"
                variant="outlined"
                density="compact"
                hide-details
                bg-color="white"
              />
              <v-select
                v-else-if="p.type === 'enum'"
                v-model="paramValues[p.name]"
                :label="p.name + (p.required ? ' *' : '')"
                :items="p.values ?? []"
                variant="outlined"
                density="compact"
                hide-details
                bg-color="white"
              />
              <v-text-field
                v-else
                v-model="paramValues[p.name]"
                :label="p.name + (p.required ? ' *' : '')"
                :type="p.type === 'date' ? 'date' : p.type === 'int' || p.type === 'float' ? 'number' : 'text'"
                :placeholder="p.default !== undefined ? String(p.default) : ''"
                variant="outlined"
                density="compact"
                hide-details
                bg-color="white"
                @keydown.enter="run"
              />
            </v-col>
          </v-row>
        </div>

        <!-- Error -->
        <v-alert v-if="runError" type="error" density="compact" variant="tonal" class="mx-4 mt-3">
          {{ runError }}
        </v-alert>

        <!-- Results toolbar -->
        <div v-if="result || running" class="results-toolbar">
          <span class="text-body-2">
            <template v-if="result">
              <v-chip v-if="result.masked_columns?.length" size="x-small" color="warning" class="mr-2">
                {{ result.masked_columns.length }} masked
              </v-chip>
              {{ result.row_count }} row{{ result.row_count !== 1 ? 's' : '' }}
              <v-chip v-if="result.truncated" size="x-small" color="warning" class="mx-1">truncated</v-chip>
              <span class="text-medium-emphasis text-caption"> · {{ result.run_id.slice(0, 8) }}</span>
            </template>
            <span v-else class="text-medium-emphasis text-caption">Running…</span>
          </span>
          <div v-if="result" class="d-flex gap-1">
            <v-btn size="x-small" variant="tonal" prepend-icon="mdi-download" @click="downloadCSV(result!, selectedQuery!.name + '.csv')">CSV</v-btn>
            <v-btn size="x-small" variant="tonal" prepend-icon="mdi-download" @click="downloadExcel(result!, selectedQuery!.name + '.xlsx')">Excel</v-btn>
          </div>
        </div>

        <!-- Results table -->
        <div class="results-scroll">
          <v-table v-if="result" density="compact" fixed-header>
            <thead>
              <tr>
                <th v-for="col in result.columns" :key="col.name" class="text-left">
                  {{ col.name }}
                  <span class="text-caption text-medium-emphasis ml-1">{{ col.type }}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, ri) in result.rows" :key="ri">
                <td v-for="(val, ci) in row" :key="ci" class="text-caption">{{ formatValue(val) }}</td>
              </tr>
            </tbody>
          </v-table>
          <div v-else-if="running" class="centered-hint">
            <v-progress-circular indeterminate size="24" class="mr-2" />Running…
          </div>
          <div v-else class="centered-hint text-medium-emphasis">
            {{ paramDefs.length ? 'Fill in parameters and press Run' : 'Press Run to execute' }}
          </div>
        </div>
      </template>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { listQueries, getQuery, type Query } from '../api/queries'
import { createRun, type RunResult, downloadCSV, downloadExcel } from '../api/runs'

const queries = ref<Query[]>([])
const loadingQueries = ref(false)
const selectedQuery = ref<Query | null>(null)
const paramDefs = ref<any[]>([])
const paramValues = reactive<Record<string, any>>({})
const result = ref<RunResult | null>(null)
const runError = ref('')
const running = ref(false)

async function load() {
  loadingQueries.value = true
  try {
    queries.value = await listQueries()
  } finally {
    loadingQueries.value = false
  }
}

async function selectQuery(q: Query) {
  selectedQuery.value = q
  result.value = null
  runError.value = ''

  // Load full query to get parameter definitions
  const full = await getQuery(q.id)
  paramDefs.value = (full.parameters ?? []).map((p: any) => ({
    ...p,
    name: p.name.replace(/^:/, ''),
  }))

  // Reset param values with defaults
  Object.keys(paramValues).forEach(k => delete paramValues[k])
  for (const p of paramDefs.value) {
    if (p.default !== undefined) paramValues[p.name] = String(p.default)
  }
}

async function run() {
  if (!selectedQuery.value) return
  running.value = true
  runError.value = ''
  result.value = null
  try {
    const cleanName = (name: string) => name.replace(/^:/, '')
    const coerced: Record<string, any> = {}
    for (const p of paramDefs.value) {
      const name = cleanName(p.name)
      const raw = paramValues[name]
      if (raw === undefined || raw === '') continue
      if (p.type === 'int') coerced[name] = parseInt(raw)
      else if (p.type === 'float') coerced[name] = parseFloat(raw)
      else if (p.type === 'bool') coerced[name] = raw === 'true'
      else coerced[name] = raw
    }
    result.value = await createRun({
      connection_id: selectedQuery.value.connection_id,
      query_id: selectedQuery.value.id,
      sql: selectedQuery.value.sql,
      parameters: Object.keys(coerced).length ? coerced : undefined,
      parameter_defs: paramDefs.value,
    })
  } catch (e: any) {
    runError.value = e.response?.data?.error ?? e.message
  } finally {
    running.value = false
  }
}

function formatValue(val: any): string {
  if (val === null || val === undefined) return 'NULL'
  if (typeof val === 'object') return JSON.stringify(val)
  return String(val)
}

onMounted(load)
</script>

<style scoped>
.home-layout {
  display: grid;
  grid-template-columns: 260px 1fr;
  height: calc(100vh - 64px);
  overflow: hidden;
}

/* Query list */
.query-list-panel {
  border-right: 1px solid rgba(0,0,0,0.08);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #888;
  padding: 14px 16px 10px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  flex-shrink: 0;
}

.query-item {
  padding: 10px 16px;
  cursor: pointer;
  border-bottom: 1px solid rgba(0,0,0,0.04);
  transition: background 0.1s;
}
.query-item:hover { background: #f5f7ff; }
.query-item.active { background: #eff3ff; border-left: 3px solid #2563eb; }

.query-item-name {
  font-size: 13px;
  font-weight: 500;
  color: #1a1a1a;
}
.query-item-desc {
  font-size: 12px;
  color: #888;
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Run panel */
.run-panel {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  height: 100%;
}

.run-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  flex-shrink: 0;
}

.run-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
}

.run-desc {
  font-size: 13px;
  color: #666;
  margin-top: 2px;
}

.params-area {
  padding: 16px 20px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  flex-shrink: 0;
  background: #fafafa;
}

.section-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: #888;
}

.results-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 20px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  background: #fafafa;
  flex-shrink: 0;
}

.results-scroll {
  flex: 1;
  overflow: auto;
}

.centered-hint {
  height: 100%;
  min-height: 200px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #aaa;
  font-size: 14px;
}

.empty-hint {
  padding: 16px;
  font-size: 13px;
  color: #aaa;
}
</style>