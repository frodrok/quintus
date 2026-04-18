<template>
  <v-container fluid class="pa-0" style="height: 100%; overflow: hidden;">
    <v-row no-gutters class="fill-height" style="height: 100%">

      <!-- Schema browser -->
      <v-col cols="2" class="panel-col">
        <div class="panel-inner">
          <div class="panel-title">Schema</div>
          <div class="schema-scroll">
            <div v-if="schemaLoading" class="empty-hint">Loading…</div>
            <div v-else-if="!form.connection_id" class="empty-hint">Select a connection</div>
            <div v-else-if="!schema.length" class="empty-hint">No tables found</div>
            <div v-else>
              <div v-for="(tables, schemaName) in schemaTree" :key="schemaName" class="schema-group">
                <div class="schema-name-row" @click="toggleSchema(String(schemaName))">
                  <v-icon size="14" class="mr-1">{{ schemaOpen[String(schemaName)] ? 'mdi-chevron-down' : 'mdi-chevron-right' }}</v-icon>
                  <span>{{ schemaName }}</span>
                </div>
                <div v-if="schemaOpen[String(schemaName)]">
                  <div v-for="t in tables" :key="t.table">
                    <div class="table-name-row" @click="insertTable(t.schema, t.table)" :title="'Insert ' + t.table">
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

      <!-- Editor panel -->
      <v-col cols="4" class="panel-col" style="border-left: 1px solid rgba(0,0,0,0.08); border-right: 1px solid rgba(0,0,0,0.08)">
        <div class="panel-inner editor-scroll">

          <!-- Header -->
          <div class="d-flex align-center gap-2 mb-3">
            <input v-model="form.name" class="name-input" placeholder="Query name" />
            <v-btn variant="tonal" size="small" :loading="saving" @click="save">Save</v-btn>
            <v-btn color="primary" size="small" :loading="running" @click="run" prepend-icon="mdi-play">Run</v-btn>
          </div>

          <!-- Connection + Description -->
          <v-row dense class="mb-1">
            <v-col cols="5">
              <v-select
                v-model="form.connection_id"
                :items="connections"
                item-title="name"
                item-value="id"
                label="Connection"
                variant="outlined"
                density="compact"
                hide-details
              />
            </v-col>
            <v-col cols="7">
              <v-text-field
                v-model="form.description"
                label="Description"
                placeholder="Optional"
                variant="outlined"
                density="compact"
                hide-details
              />
            </v-col>
          </v-row>

          <!-- SQL -->
          <div class="mb-2" style="flex-shrink: 0">
            <div class="section-label mb-1">SQL</div>
            <textarea
              v-model="form.sql"
              class="sql-input"
              spellcheck="false"
              placeholder="SELECT … WHERE city = :city"
              @keydown.ctrl.enter.prevent="run"
              @keydown.meta.enter.prevent="run"
            />
          </div>

          <!-- Parameters -->
          <div class="section-box mb-2">
            <div class="section-box-header">
              <span class="section-label">Parameters</span>
              <v-btn size="x-small" variant="text" @click="addParam" prepend-icon="mdi-plus">Add</v-btn>
            </div>
            <div v-for="(p, i) in form.parameters" :key="i" class="param-row">
              <input v-model="p.name" placeholder=":name" class="mini-input" style="width:100px" />
              <select v-model="p.type" class="mini-select">
                <option v-for="t in ['string','int','float','date','bool','enum']" :key="t" :value="t">{{ t }}</option>
              </select>
              <input v-model="p.default" placeholder="default" class="mini-input" style="width:70px" />
              <label class="req-label"><input type="checkbox" v-model="p.required" /> req</label>
              <input v-if="p.type === 'enum'" v-model="p.valuesStr" placeholder="a,b,c" class="mini-input" style="width:70px" />
              <v-btn icon="mdi-close" size="x-small" variant="text" color="error" @click="removeParam(i)" />
            </div>
            <div v-if="!form.parameters.length" class="empty-hint">No parameters. Use :name in SQL.</div>
          </div>

          <!-- Column masks -->
          <div class="section-box mb-2">
            <div class="section-box-header">
              <span class="section-label">Column masks</span>
              <v-btn size="x-small" variant="text" @click="addMask" prepend-icon="mdi-plus">Add</v-btn>
            </div>
            <div v-for="(m, i) in form.column_masks" :key="i" class="param-row">
              <input v-model="m.column" placeholder="column" class="mini-input" style="width:110px" />
              <select v-model="m.mask" class="mini-select">
                <option v-for="t in ['redacted','partial','null','hash']" :key="t" :value="t">{{ t }}</option>
              </select>
              <input
                :value="maskGroups(i)"
                @input="setMaskGroups(i, ($event.target as HTMLInputElement).value)"
                placeholder="groups (comma separated)"
                class="mini-input"
                style="flex:1; min-width:0"
              />
              <v-btn icon="mdi-close" size="x-small" variant="text" color="error" @click="removeMask(i)" />
            </div>
            <div v-if="!form.column_masks.length" class="empty-hint">No masks defined.</div>
          </div>

          <!-- Row mask -->
          <div class="section-box mb-2">
            <div class="section-box-header">
              <span class="section-label">Row mask</span>
              <v-btn v-if="!form.row_mask" size="x-small" variant="text" @click="addRowMask" prepend-icon="mdi-plus">Add</v-btn>
              <v-btn v-else size="x-small" variant="text" color="error" @click="removeRowMask">Remove</v-btn>
            </div>
            <div v-if="form.row_mask" class="param-row">
              <input v-model="form.row_mask.condition_column" placeholder="condition column (e.g. is_female)" class="mini-input" style="flex:1; min-width:0" />
              <input
                :value="form.row_mask.visible_to_groups.join(',')"
                @input="form.row_mask.visible_to_groups = ($event.target as HTMLInputElement).value.split(',').map(s => s.trim()).filter(Boolean)"
                placeholder="visible to groups"
                class="mini-input"
                style="flex:1; min-width:0"
              />
            </div>
            <div v-else class="empty-hint">No row mask defined.</div>
          </div>

          <v-alert v-if="saveError" type="error" density="compact" variant="tonal">{{ saveError }}</v-alert>
        </div>
      </v-col>

      <!-- Results panel -->
      <v-col cols="6" class="panel-col">
        <div class="panel-inner">

          <!-- Parameter inputs -->
          <div v-if="form.parameters.length" class="param-inputs-area">
            <div class="section-label mb-2">Parameters</div>
            <v-row dense>
              <v-col v-for="p in form.parameters" :key="p.name" cols="6">
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
                  :placeholder="p.default !== undefined ? String(p.default) : p.name"
                  variant="outlined"
                  density="compact"
                  hide-details
                  bg-color="white"
                />
              </v-col>
            </v-row>
          </div>

          <!-- Results toolbar -->
          <div class="results-toolbar">
            <span class="text-body-2">
              <template v-if="result">
                <v-chip v-if="result.masked_columns?.length" size="x-small" color="warning" class="mr-2">{{ result.masked_columns.length }} masked</v-chip>
                {{ result.row_count }} row{{ result.row_count !== 1 ? 's' : '' }}
                <v-chip v-if="result.truncated" size="x-small" color="warning" class="mx-1">truncated</v-chip>
                <span class="text-medium-emphasis text-caption"> · {{ result.run_id.slice(0, 8) }}</span>
              </template>
              <span v-else-if="runError" class="text-error text-caption">{{ runError }}</span>
              <span v-else class="text-medium-emphasis text-caption">No results yet</span>
            </span>
            <div v-if="result" class="d-flex gap-1">
              <v-btn size="x-small" variant="tonal" prepend-icon="mdi-download" @click="downloadCSV(result!, form.name + '.csv')">CSV</v-btn>
              <v-btn size="x-small" variant="tonal" prepend-icon="mdi-download" @click="downloadExcel(result!, form.name + '.xlsx')">Excel</v-btn>
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
              <v-progress-circular indeterminate size="20" class="mr-2" />Running…
            </div>
            <div v-else class="centered-hint">Run a query to see results.</div>
          </div>

        </div>
      </v-col>

    </v-row>
  </v-container>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getQuery, createQuery, updateQuery, type Parameter, type ColumnMask, type RowMask } from '../api/queries'
import { createRun, type RunResult, downloadCSV, downloadExcel } from '../api/runs'
import { listConnections, type Connection } from '../api/connections'
import { getConnectionSchema, type SchemaTable } from '../api/connections'

const route = useRoute()
const router = useRouter()
const isNew = computed(() => route.path === '/queries/new')
const queryId = computed(() => isNew.value ? null : route.params.id as string)

const connections = ref<Connection[]>([])
const result = ref<RunResult | null>(null)
const runError = ref('')
const saveError = ref('')
const saving = ref(false)
const running = ref(false)
const paramValues = reactive<Record<string, any>>({})

const schema = ref<SchemaTable[]>([])
const schemaOpen = ref<Record<string, boolean>>({})
const schemaLoading = ref(false)

type ParamForm = Parameter & { valuesStr?: string }

const form = reactive({
  name: '',
  description: '',
  connection_id: '',
  sql: '',
  parameters: [] as ParamForm[],
  column_masks: [] as ColumnMask[],
  row_mask: null as RowMask | null,
})

const schemaTree = computed(() => {
  const tree: Record<string, SchemaTable[]> = {}
  for (const t of schema.value) {
    if (!tree[t.schema]) tree[t.schema] = []
    tree[t.schema].push(t)
  }
  return tree
})

function toggleSchema(name: string) {
  schemaOpen.value[name] = !schemaOpen.value[name]
}

watch(() => form.connection_id, async (id) => {
  schema.value = []
  schemaOpen.value = {}
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

function insertTable(schemaName: string, table: string) {
  const insert = schemaName === 'public' ? table : `${schemaName}.${table}`
  form.sql = form.sql ? form.sql + '\n' + insert : insert
}

function addParam() { form.parameters.push({ name: '', type: 'string', required: false }) }
function removeParam(i: number) { form.parameters.splice(i, 1) }
function addMask() { form.column_masks.push({ column: '', visible_to_groups: [], mask: 'redacted' }) }
function removeMask(i: number) { form.column_masks.splice(i, 1) }
function maskGroups(i: number): string { return form.column_masks[i].visible_to_groups.join(',') }
function setMaskGroups(i: number, val: string) {
  form.column_masks[i].visible_to_groups = val.split(',').map(s => s.trim()).filter(Boolean)
}
function addRowMask() { form.row_mask = { condition_column: '', visible_to_groups: [] } }
function removeRowMask() { form.row_mask = null }

async function load() {
  connections.value = await listConnections()
  if (!isNew.value && queryId.value) {
    const q = await getQuery(queryId.value)
    form.name = q.name
    form.description = q.description ?? ''
    form.connection_id = q.connection_id
    form.sql = q.sql
    form.parameters = (q.parameters ?? []).map(p => ({ ...p, valuesStr: p.values?.join(',') ?? '' }))
    form.column_masks = q.column_masks ?? []
    form.row_mask = q.row_mask ?? null
  }
}

async function save() {
  if (!form.name || !form.connection_id || !form.sql) {
    saveError.value = 'Name, connection, and SQL are required.'
    return
  }
  saving.value = true
  saveError.value = ''
  try {
    const parameters: Parameter[] = form.parameters.map(p => ({
      name: p.name,
      type: p.type,
      required: p.required,
      default: p.default,
      values: p.type === 'enum' && p.valuesStr ? p.valuesStr.split(',').map(s => s.trim()) : undefined,
    }))
    const input = {
      name: form.name,
      description: form.description || undefined,
      connection_id: form.connection_id,
      sql: form.sql,
      parameters,
      column_masks: form.column_masks,
      row_mask: form.row_mask ?? undefined,
    }
    if (isNew.value) {
      const q = await createQuery(input)
      router.replace(`/queries/${q.id}/edit`)
    } else {
      await updateQuery(queryId.value!, input)
    }
  } catch (e: any) {
    saveError.value = e.response?.data?.error ?? e.message
  } finally {
    saving.value = false
  }
}

async function run() {
  if (!form.connection_id || !form.sql) {
    runError.value = 'Select a connection and enter SQL first.'
    return
  }
  running.value = true
  runError.value = ''
  result.value = null
  try {
    const cleanName = (name: string) => name.replace(/^:/, '')
    const coerced: Record<string, any> = {}
    for (const p of form.parameters) {
      const name = cleanName(p.name)
      const raw = paramValues[p.name] ?? paramValues[name]
      if (raw === undefined || raw === '') continue
      if (p.type === 'int') coerced[name] = parseInt(raw)
      else if (p.type === 'float') coerced[name] = parseFloat(raw)
      else if (p.type === 'bool') coerced[name] = raw === 'true'
      else coerced[name] = raw
    }
    result.value = await createRun({
      connection_id: form.connection_id,
      query_id: queryId.value ?? undefined,
      sql: form.sql,
      parameters: Object.keys(coerced).length ? coerced : undefined,
      parameter_defs: form.parameters.map(p => ({
        name: cleanName(p.name),
        type: p.type,
        required: p.required,
        default: p.default,
        values: p.values,
      })),
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
.editor-scroll :deep(.v-input) {
  margin-bottom: 0 !important;
}

.editor-scroll :deep(.v-messages) {
  display: none;
}
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

/* Schema */
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
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.table-name-row:hover { background: #eff6ff; border-radius: 4px; }

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

/* Editor */
.editor-scroll {
  overflow-y: auto;
  padding: 16px;
  height: 100%;
  box-sizing: border-box;
}
.name-input {
  flex: 1;
  font-size: 17px;
  font-weight: 600;
  border: none;
  border-bottom: 2px solid #e5e7eb;
  padding: 3px 0;
  outline: none;
  background: transparent;
  min-width: 0;
}
.name-input:focus { border-bottom-color: #2563eb; }

.section-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: #888;
}

.sql-input {
  width: 100%;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 13px;
  padding: 10px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  resize: vertical;
  line-height: 1.6;
  min-height: 180px;
  background: #fafafa;
  color: #1a1a1a;
  box-sizing: border-box;
}
.sql-input:focus {
  outline: none;
  border-color: #2563eb;
  background: #fff;
}

.section-box {
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  overflow: hidden;
}

.section-box-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  background: #fafafa;
  border-bottom: 1px solid #e5e7eb;
}

.param-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-bottom: 1px solid #f3f4f6;
}
.param-row:last-child { border-bottom: none; }

.mini-input {
  padding: 4px 7px;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  font-size: 12px;
  background: #fff;
  color: #1a1a1a;
  min-width: 0;
}
.mini-input:focus { outline: none; border-color: #2563eb; }

.mini-select {
  padding: 4px 6px;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  font-size: 12px;
  background: #fff;
  color: #1a1a1a;
}

.req-label {
  font-size: 11px;
  color: #666;
  display: flex;
  align-items: center;
  gap: 3px;
  white-space: nowrap;
  flex-shrink: 0;
}

.empty-hint {
  font-size: 12px;
  color: #aaa;
  padding: 8px 12px;
}

/* Results */
.param-inputs-area {
  padding: 12px 16px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  background: #fafafa;
  flex-shrink: 0;
}

.results-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  border-bottom: 1px solid rgba(0,0,0,0.07);
  background: #fafafa;
  flex-shrink: 0;
}

.results-scroll {
  flex: 1;
  overflow: auto;
}

.centered-hint {
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #aaa;
  font-size: 14px;
}
</style>