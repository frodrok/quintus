<template>
  <v-container fluid class="pa-0 fill-height">
    <v-row no-gutters class="fill-height">

      <!-- Schema browser -->
      <v-col cols="2" class="panel-col">
        <div class="panel-inner">
          <div class="panel-title">Schema</div>
          <div class="schema-scroll">
            <div v-if="!selectedConnectionId" class="empty-hint">Select a connection</div>
            <div v-else-if="schemaLoading" class="empty-hint">Loading…</div>
            <div v-else-if="!schema.length" class="empty-hint">No tables found</div>
            <div v-else>
              <div v-for="(tables, schemaName) in schemaTree" :key="schemaName" class="schema-group">
                <div class="schema-name-row" @click="toggleSchema(String(schemaName))">
                  <v-icon size="14" class="mr-1">{{ schemaOpen[String(schemaName)] ? 'mdi-chevron-down' : 'mdi-chevron-right' }}</v-icon>
                  <span>{{ schemaName }}</span>
                </div>
                <div v-if="schemaOpen[String(schemaName)]">
                  <div v-for="t in tables" :key="t.table">
                    <div class="table-name-row">
                      <v-icon size="12" class="mr-1" color="primary">mdi-table</v-icon>
                      <span>{{ t.table }}</span>
                    </div>
                    <div class="col-list">
                      <div v-for="col in t.columns" :key="col.name" class="col-row">
                        <span class="col-name">{{ col.name }}</span>
                        <span class="col-type">{{ col.data_type }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </v-col>

      <!-- Main panel -->
      <v-col cols="10" class="panel-col" style="border-left: 1px solid rgba(0,0,0,0.08)">
        <div class="panel-inner">

          <!-- Toolbar -->
          <div class="explore-toolbar">
            <v-select
              v-model="selectedConnectionId"
              :items="connections"
              item-title="name"
              item-value="id"
              placeholder="Select connection"
              variant="outlined"
              density="compact"
              hide-details
              style="max-width: 220px"
            />
            <v-spacer />
            <div v-if="result" class="d-flex gap-1">
              <v-btn size="x-small" variant="tonal" prepend-icon="mdi-download" @click="downloadCSV(result!, question + '.csv')">CSV</v-btn>
              <v-btn size="x-small" variant="tonal" prepend-icon="mdi-download" @click="downloadExcel(result!, question + '.xlsx')">Excel</v-btn>
            </div>
          </div>

          <!-- AI question input -->
          <div class="question-area">
            <div class="question-input-row">
              <v-text-field
                v-model="question"
                placeholder="Visa alla kunder från Stockholm som köpt något senaste månaden…"
                variant="outlined"
                density="compact"
                hide-details
                :disabled="!selectedConnectionId || generatingSql"
                @keydown.enter="askAi"
                class="flex-grow-1"
              >
                <template #prepend-inner>
                  <v-icon color="purple" class="mr-1">mdi-creation</v-icon>
                </template>
              </v-text-field>
              <v-btn
                color="purple"
                :loading="generatingSql"
                :disabled="!selectedConnectionId || !question.trim()"
                @click="askAi"
                class="ml-2"
              >
                Ask AI
              </v-btn>
            </div>
            <div v-if="aiError" class="text-caption text-error mt-1 ml-1">{{ aiError }}</div>
          </div>

          <!-- Generated SQL -->
          <div v-if="generatedSql" class="sql-area">
            <div class="sql-label">
              <v-icon size="14" class="mr-1" color="grey">mdi-code-tags</v-icon>
              Generated SQL
              <v-btn
                size="x-small"
                variant="text"
                class="ml-2"
                @click="editingSql = !editingSql"
              >
                {{ editingSql ? 'Done' : 'Edit' }}
              </v-btn>
              <v-btn
                size="x-small"
                color="primary"
                variant="tonal"
                class="ml-1"
                :loading="running"
                @click="run"
                prepend-icon="mdi-play"
              >
                Run
              </v-btn>
            </div>
            <textarea
              v-if="editingSql"
              v-model="generatedSql"
              class="sql-input"
              spellcheck="false"
              @keydown.ctrl.enter.prevent="run"
              @keydown.meta.enter.prevent="run"
            />
            <pre v-else class="sql-preview">{{ generatedSql }}</pre>
          </div>

          <!-- Results toolbar -->
          <div v-if="result || running || runError" class="results-toolbar">
            <template v-if="result">
              <v-chip v-if="result.masked_columns?.length" size="x-small" color="warning" class="mr-2">
                {{ result.masked_columns.length }} masked
              </v-chip>
              <span class="text-body-2">
                {{ result.row_count }} row{{ result.row_count !== 1 ? 's' : '' }}
                <v-chip v-if="result.truncated" size="x-small" color="warning" class="mx-1">truncated</v-chip>
                <span class="text-medium-emphasis text-caption"> · {{ result.run_id.slice(0, 8) }}</span>
              </span>
            </template>
            <span v-else-if="runError" class="text-error text-caption">{{ runError }}</span>
            <span v-else class="text-medium-emphasis text-caption">Running…</span>
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
            <div v-else-if="!generatedSql" class="centered-hint">
              <div class="text-center">
                <v-icon size="48" color="purple-lighten-3">mdi-creation</v-icon>
                <div class="mt-3 text-medium-emphasis">
                  Select a connection and describe what you want to see
                </div>
              </div>
            </div>
          </div>

        </div>
      </v-col>

    </v-row>
  </v-container>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { listConnections, getConnectionSchema, type Connection, type SchemaTable } from '../api/connections'
import { createRun, type RunResult, downloadCSV, downloadExcel } from '../api/runs'

const OLLAMA_URL = 'http://localhost:11434'
const OLLAMA_MODEL = 'qwen2.5-coder:7b'

const connections = ref<Connection[]>([])
const selectedConnectionId = ref<string>('')
const schema = ref<SchemaTable[]>([])
const schemaOpen = ref<Record<string, boolean>>({})
const schemaLoading = ref(false)

const question = ref('')
const generatedSql = ref('')
const editingSql = ref(false)
const generatingSql = ref(false)
const aiError = ref('')

const result = ref<RunResult | null>(null)
const runError = ref('')
const running = ref(false)

const schemaTree = computed(() => {
  const tree: Record<string, SchemaTable[]> = {}
  for (const t of schema.value) {
    if (!tree[t.schema]) tree[t.schema] = []
    tree[t.schema].push(t)
  }
  return tree
})

// Build schema context string for the AI prompt
const schemaContext = computed(() => {
  const parts: string[] = []
  for (const [schemaName, tables] of Object.entries(schemaTree.value)) {
    for (const t of tables as SchemaTable[]) {
      if (t.table.startsWith('ƒ')) continue // skip functions
      const cols = t.columns.map(c => `${c.name} ${c.data_type}`).join(', ')
      const tableName = schemaName === 'public' ? t.table : `${schemaName}.${t.table}`
      parts.push(`${tableName}(${cols})`)
    }
  }
  return parts.join('\n')
})

watch(selectedConnectionId, async (id) => {
  schema.value = []
  schemaOpen.value = {}
  result.value = null
  generatedSql.value = ''
  runError.value = ''
  aiError.value = ''
  if (!id) return
  schemaLoading.value = true
  try {
    schema.value = await getConnectionSchema(id)
    const first = schema.value[0]?.schema
    if (first) schemaOpen.value = { [first]: true }
  } catch {
  } finally {
    schemaLoading.value = false
  }
})

function toggleSchema(name: string) {
  schemaOpen.value[name] = !schemaOpen.value[name]
}

async function askAi() {
  if (!question.value.trim() || !schemaContext.value) return
  generatingSql.value = true
  aiError.value = ''
  generatedSql.value = ''
  result.value = null
  runError.value = ''
  editingSql.value = false

  const prompt = `You are a PostgreSQL expert. Given these tables:
${schemaContext.value}

Write a PostgreSQL SELECT query for: "${question.value}"

Rules:
- Return ONLY the SQL query, no explanation, no markdown, no backticks
- Use only the tables and columns listed above
- Always add LIMIT 1000 unless the user asks for all data
- Use proper PostgreSQL syntax`

  try {
    const response = await fetch(`${OLLAMA_URL}/api/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: OLLAMA_MODEL,
        prompt,
        stream: false,
      }),
    })

    if (!response.ok) throw new Error(`Ollama error: ${response.status}`)

    const data = await response.json()
    let sql = data.response?.trim() ?? ''

    // Strip markdown code fences if model adds them anyway
    sql = sql.replace(/^```sql\n?/i, '').replace(/^```\n?/, '').replace(/\n?```$/, '').trim()

    generatedSql.value = sql

    // Auto-run
    await run()
  } catch (e: any) {
    aiError.value = e.message.includes('Failed to fetch')
      ? 'Cannot reach Ollama at localhost:11434. Is it running?'
      : e.message
  } finally {
    generatingSql.value = false
  }
}

async function run() {
  if (!selectedConnectionId.value || !generatedSql.value.trim()) return
  running.value = true
  runError.value = ''
  result.value = null
  try {
    result.value = await createRun({
      connection_id: selectedConnectionId.value,
      sql: generatedSql.value,
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

onMounted(async () => {
  connections.value = await listConnections()
  if (connections.value.length === 1) {
    selectedConnectionId.value = connections.value[0].id
  }
})
</script>

<style scoped>
.panel-col {
  height: calc(100vh - 64px);
  overflow: hidden;
}

.panel-inner {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.panel-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #888;
  padding: 10px 12px 8px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  flex-shrink: 0;
}

.schema-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.schema-group { margin-bottom: 2px; }

.schema-name-row {
  display: flex;
  align-items: center;
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 600;
  color: #444;
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.schema-name-row:hover { background: #f3f4f6; border-radius: 4px; }

.table-name-row {
  display: flex;
  align-items: center;
  padding: 3px 10px 3px 20px;
  font-size: 12px;
  color: #2563eb;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.col-list { padding-left: 28px; }
.col-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1px 8px;
  font-size: 11px;
}
.col-name { color: #555; }
.col-type {
  color: #aaa;
  background: #f3f4f6;
  border-radius: 2px;
  padding: 0 3px;
  font-size: 10px;
  flex-shrink: 0;
  margin-left: 4px;
}

.explore-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  background: #fafafa;
  flex-shrink: 0;
}

.question-area {
  padding: 14px 16px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  flex-shrink: 0;
  background: #faf5ff;
}

.question-input-row {
  display: flex;
  align-items: center;
}

.sql-area {
  padding: 10px 16px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  flex-shrink: 0;
  background: #f8faff;
}

.sql-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: #888;
  margin-bottom: 6px;
  display: flex;
  align-items: center;
}

.sql-preview {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 12px;
  color: #1a1a1a;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  padding: 10px 12px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.sql-input {
  width: 100%;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 12px;
  padding: 10px 12px;
  border: 1px solid #2563eb;
  border-radius: 6px;
  resize: vertical;
  line-height: 1.6;
  min-height: 80px;
  background: #fff;
  color: #1a1a1a;
  box-sizing: border-box;
}
.sql-input:focus { outline: none; }

.results-toolbar {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  background: #fafafa;
  flex-shrink: 0;
  min-height: 38px;
}

.results-scroll {
  flex: 1;
  overflow: auto;
}

.centered-hint {
  height: 100%;
  min-height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #aaa;
  font-size: 14px;
}

.empty-hint {
  padding: 12px;
  font-size: 12px;
  color: #aaa;
}
</style>