<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { useToast } from '@/components/ui/toast'
import { TOAST_DURATION } from '@/lib/constants'
import { useSourcesStore } from '@/stores/sources'
import { Code, ChevronsUpDown, Plus, Database } from 'lucide-vue-next'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Separator } from '@/components/ui/separator'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog'

const router = useRouter()
const { toast } = useToast()
const sourcesStore = useSourcesStore()

// Define types for our API requests and responses
interface ConnectionInfo {
    host: string;
    username: string;
    password: string;
    database: string;
    table_name: string;
}

// Form state
const tableMode = ref<'create' | 'connect'>('create') // 'create' or 'connect'
const createTable = computed(() => tableMode.value === 'create')
const sourceName = ref<string | number>('')
const host = ref<string | number>('')
const enableAuth = ref<boolean>(false)
const username = ref<string | number>('')
const password = ref<string | number>('')
const database = ref<string | number>('')
const tableName = ref<string | number>('')
const description = ref<string | number>('')
const ttlDays = ref<string | number>(90)
const metaTSField = ref<string | number>('timestamp')
const metaSeverityField = ref<string | number>('severity_text')

// Use connection validation with source store
const isValidating = ref(false)
const validationResult = ref(null)
const isValidated = ref(false)

// Validate connection using source store
const validateConnection = async (connectionInfo) => {
    isValidating.value = true
    isValidated.value = false
    validationResult.value = null

    try {
        const result = await sourcesStore.validateSourceConnection(connectionInfo)

        if (result.success && result.data) {
            validationResult.value = result.data
            isValidated.value = true
        }
    } catch (error) {
        console.error("Validation error:", error)
    } finally {
        isValidating.value = false
    }
}

// Schema preview
const tableSchema = `CREATE TABLE IF NOT EXISTS {{database_name}}.{{table_name}}
(
    timestamp DateTime64(3) CODEC(DoubleDelta, LZ4),
    trace_id String CODEC(ZSTD(1)),
    span_id String CODEC(ZSTD(1)),
    trace_flags UInt32 CODEC(ZSTD(1)),
    severity_text LowCardinality(String) CODEC(ZSTD(1)),
    severity_number Int32 CODEC(ZSTD(1)),
    service_name LowCardinality(String) CODEC(ZSTD(1)),
    namespace LowCardinality(String) CODEC(ZSTD(1)),
    body String CODEC(ZSTD(1)),
    log_attributes Map(LowCardinality(String), String) CODEC(ZSTD(1)),

    INDEX idx_trace_id trace_id TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_severity_text severity_text TYPE set(100) GRANULARITY 4,
    INDEX idx_log_attributes_keys mapKeys(log_attributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_log_attributes_values mapValues(log_attributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_body body TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY toDate(timestamp)
ORDER BY (namespace, service_name, timestamp)
TTL toDateTime(timestamp) + INTERVAL {{ttl_day}} DAY
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;`

// Schema state
const isEditingSchema = ref(false)
const editedSchema = ref('')

// Function to generate the default schema
function generateSchema() {
    const db = database.value ? String(database.value) : 'your_database'
    const table = tableName.value ? String(tableName.value) : 'your_table'
    let schema = tableSchema

    // Replace placeholders with actual values
    schema = schema.replace(/{{database_name}}/g, db)
    schema = schema.replace(/{{table_name}}/g, table)
    schema = schema.replace(/{{ttl_day}}/g, String(ttlDays.value))

    return schema
}

// Watch for changes and update schema
watch([database, tableName, ttlDays], () => {
    // Always regenerate the schema with current values
    editedSchema.value = generateSchema()
}, { immediate: true })

// Computed property for the actual schema to use
const actualSchema = computed(() => {
    return editedSchema.value || generateSchema()
})

// Function to reset schema to default
function resetSchema() {
    editedSchema.value = generateSchema()
    isEditingSchema.value = false
}

// Form validation
const isValid = computed(() => {
    if (!sourceName.value || !host.value || !database.value || !tableName.value) return false
    if (enableAuth.value && (!username.value || !password.value)) return false
    return true
})

// Event handlers
const handleAuthToggle = (checked: boolean) => {
    enableAuth.value = checked
}

// Computed properties
const validateButtonText = computed(() => {
    if (isValidating.value) return 'Validating...'
    return tableMode.value === 'connect' ? 'Validate Connection & Columns' : 'Validate Connection'
})

const submitButtonText = computed(() => {
    if (isSubmitting.value) return 'Creating...'
    if (createTable.value) return 'Create Source'

    // For "Connect Existing Table" mode
    if (isValidated.value) return 'Import Source'
    return 'Validate & Import'
})

// Validate connection handler
const handleValidateConnection = async () => {
    // Prepare connection info
    const connectionInfo: ConnectionInfo = {
        host: String(host.value),
        username: enableAuth.value ? String(username.value) : '',
        password: enableAuth.value ? String(password.value) : '',
        database: String(database.value),
        table_name: String(tableName.value),
    }

    // Add timestamp and severity fields if connecting to existing table
    if (tableMode.value === 'connect' && tableName.value) {
        connectionInfo.timestamp_field = String(metaTSField.value)
        // Only add severity field if it's not empty
        if (metaSeverityField.value) {
            connectionInfo.severity_field = String(metaSeverityField.value)
        }
    }

    await validateConnection(connectionInfo)
}

// Form state
const isSubmitting = ref(false)
const formError = ref(null)

const submitForm = async () => {
    if (!isValid.value) {
        toast({
            title: 'Error',
            description: 'Please fill in all required fields',
            variant: 'destructive',
            duration: TOAST_DURATION.ERROR,
        })
        return
    }

    // For "Connect Existing Table" mode, validate first if not already validated
    if (!createTable.value && !isValidated.value) {
        await handleValidateConnection()
        // If validation failed, don't proceed
        if (!isValidated.value) return
    }

    isSubmitting.value = true

    try {
        // Create payload with proper types
        const payload = {
            name: String(sourceName.value),
            meta_is_auto_created: createTable.value,
            meta_ts_field: String(metaTSField.value),
            meta_severity_field: metaSeverityField.value ? String(metaSeverityField.value) : "",
            connection: {
                host: String(host.value),
                username: enableAuth.value ? String(username.value) : '',
                password: enableAuth.value ? String(password.value) : '',
                database: String(database.value),
                table_name: String(tableName.value),
            },
            description: String(description.value),
            ttl_days: Number(ttlDays.value),
            schema: createTable.value ? actualSchema.value : undefined,
        }

        // Use the store directly
        const result = await sourcesStore.createSource(payload)

        if (result.success) {
            // Redirect to sources list
            router.push({ name: 'Sources' })
        } else {
            formError.value = result.error
        }
    } catch (error) {
        console.error("Error creating source:", error)
        formError.value = error instanceof Error ? error.message : 'Unknown error'
    } finally {
        isSubmitting.value = false
    }
}
</script>

<template>
    <div class="container mx-auto max-w-4xl px-4 py-8">
        <Card>
            <CardHeader>
                <CardTitle>Add Source</CardTitle>
                <CardDescription>
                    Connect to a ClickHouse database and configure log ingestion
                </CardDescription>
            </CardHeader>
            <CardContent>
                <form @submit.prevent="submitForm" class="space-y-6">
                    <div class="space-y-6">
                        <!-- Basic Info -->
                        <div class="space-y-4">
                            <div class="flex items-center justify-between">
                                <h3 class="text-lg font-medium">Basic Information</h3>
                                <div class="text-sm text-muted-foreground">
                                    Define your source details
                                </div>
                            </div>

                            <div class="grid gap-2">
                                <Label for="source_name" class="required">Source Name</Label>
                                <Input id="source_name" v-model="sourceName" placeholder="My Application Logs"
                                    maxlength="50" required />
                                <p class="text-sm text-muted-foreground">
                                    A descriptive name (up to 50 characters) to identify this data source
                                </p>
                            </div>

                            <div class="grid gap-2">
                                <Label for="description">Description</Label>
                                <Textarea id="description" v-model="description"
                                    placeholder="Optional description of what this source contains" rows="2"
                                    maxlength="500" />
                                <p class="text-sm text-muted-foreground">
                                    Optional: Add details about what kind of logs this source will contain
                                </p>
                            </div>
                        </div>

                        <!-- Connection Details -->
                        <div class="space-y-4">
                            <div class="flex items-center justify-between">
                                <h3 class="text-lg font-medium">Connection Details</h3>
                                <div class="text-sm text-muted-foreground">
                                    Configure ClickHouse connection
                                </div>
                            </div>

                            <div class="grid gap-2">
                                <Label for="host" class="required">Host and Port</Label>
                                <Input id="host" v-model="host" placeholder="localhost:9000" required />
                                <p class="text-sm text-muted-foreground">
                                    Enter the ClickHouse server host and port in the format host:port (e.g.,
                                    localhost:9000). Port 9000 is the default TCP protocol port used by ClickHouse.
                                </p>
                            </div>

                            <!-- Database and Table Name side by side -->
                            <div class="grid grid-cols-2 gap-4">
                                <div class="grid gap-2">
                                    <Label for="database" class="required">Database</Label>
                                    <Input id="database" v-model="database" placeholder="default" required />
                                </div>

                                <div class="grid gap-2">
                                    <Label for="table_name" class="required">Table Name</Label>
                                    <Input id="table_name" v-model="tableName" placeholder="app_logs" required />
                                </div>
                            </div>
                            <p class="text-sm text-muted-foreground">
                                The database and table where your logs will be stored in ClickHouse
                            </p>

                            <!-- Auth Toggle -->
                            <div class="space-y-4">
                                <div class="flex items-center justify-between bg-muted/50 p-3 rounded-md">
                                    <div class="space-y-0.5">
                                        <Label class="text-base">Authentication</Label>
                                        <p class="text-sm text-muted-foreground">
                                            Enable if your ClickHouse server requires authentication
                                        </p>
                                    </div>
                                    <Switch :checked="enableAuth" @update:checked="handleAuthToggle" />
                                </div>

                                <!-- Auth Fields - Only shown when authentication is enabled -->
                                <div v-show="enableAuth"
                                    class="grid gap-4 md:grid-cols-2 border-l-2 border-primary/20 pl-3 animate-in fade-in slide-in-from-top-2">
                                    <div class="grid gap-2">
                                        <Label for="username" class="required">Username</Label>
                                        <Input id="username" v-model="username" placeholder="default"
                                            :required="enableAuth" />
                                    </div>

                                    <div class="grid gap-2">
                                        <Label for="password" class="required">Password</Label>
                                        <Input id="password" v-model="password" type="password"
                                            :required="enableAuth" />
                                    </div>
                                </div>
                            </div>
                        </div>

                        <!-- Table Configuration Option -->
                        <div class="space-y-4">
                            <div class="flex items-center justify-between">
                                <h3 class="text-lg font-medium">Table Configuration</h3>
                                <div class="text-sm text-muted-foreground">
                                    Choose how to handle your log data table
                                </div>
                            </div>

                            <RadioGroup :model-value="tableMode"
                                @update:model-value="(val) => tableMode = val as 'create' | 'connect'"
                                class="grid grid-cols-[1fr_auto_1fr] items-start gap-4">
                                <!-- Create Table Card -->
                                <Card
                                    :class="{ 'border-primary shadow-sm': tableMode === 'create', 'border-muted-foreground/20': tableMode !== 'create' }"
                                    class="cursor-pointer transition-all hover:border-primary/70"
                                    @click="tableMode = 'create' as const">
                                    <CardHeader>
                                        <div class="flex items-center gap-2">
                                            <RadioGroupItem value="create" id="create" />
                                            <Label for="create" class="cursor-pointer font-medium">Create New
                                                Table</Label>
                                        </div>
                                    </CardHeader>
                                    <CardContent class="space-y-4">
                                        <div class="flex items-start gap-4">
                                            <Plus class="h-5 w-5 mt-1 text-muted-foreground" />
                                            <div class="space-y-1">
                                                <p class="text-sm font-medium">Let LogChef create the table</p>
                                                <p class="text-sm text-muted-foreground">Optimized schema with
                                                    OTLP-compatible fields and efficient compression</p>
                                            </div>
                                        </div>

                                        <!-- TTL Days for Create Table -->
                                        <div class="grid gap-2 mt-4 border-t pt-4">
                                            <Label for="ttl_days">TTL Days</Label>
                                            <Input id="ttl_days" v-model="ttlDays" type="number" min="1" />
                                            <p class="text-sm text-muted-foreground">
                                                Number of days to keep logs before automatic deletion
                                            </p>
                                        </div>

                                        <!-- Schema Preview -->
                                        <Dialog>
                                            <DialogTrigger asChild>
                                                <Button variant="outline"
                                                    class="w-full flex items-center justify-between">
                                                    <div class="flex items-center gap-2">
                                                        <Code class="h-4 w-4" />
                                                        <span>View Auto-Generated Schema</span>
                                                    </div>
                                                    <ChevronsUpDown class="h-4 w-4" />
                                                </Button>
                                            </DialogTrigger>
                                            <DialogContent class="sm:max-w-[800px]">
                                                <DialogHeader>
                                                    <DialogTitle>Table Schema</DialogTitle>
                                                    <DialogDescription>
                                                        This schema will be used to create your table. You can edit it
                                                        if needed.
                                                    </DialogDescription>
                                                </DialogHeader>

                                                <div class="space-y-4 py-4">
                                                    <div class="flex items-center justify-between">
                                                        <div class="space-y-1">
                                                            <h4 class="text-sm font-medium leading-none">Schema
                                                                Definition</h4>
                                                            <p class="text-sm text-muted-foreground">
                                                                The CREATE TABLE statement that will be executed
                                                            </p>
                                                        </div>
                                                        <div class="flex items-center gap-2">
                                                            <Button variant="outline" size="sm" @click="resetSchema"
                                                                :disabled="!isEditingSchema">
                                                                Reset to Default
                                                            </Button>
                                                            <Button variant="outline" size="sm"
                                                                @click="isEditingSchema = !isEditingSchema">
                                                                {{ isEditingSchema ? 'Preview' : 'Edit' }}
                                                            </Button>
                                                        </div>
                                                    </div>

                                                    <!-- Schema Content -->
                                                    <div v-if="!isEditingSchema" class="rounded-md bg-muted p-4">
                                                        <pre
                                                            class="text-sm text-muted-foreground whitespace-pre-wrap">{{ actualSchema }}</pre>
                                                    </div>
                                                    <Textarea v-else v-model="editedSchema"
                                                        :placeholder="generateSchema()" class="font-mono text-sm"
                                                        rows="20" />
                                                </div>
                                            </DialogContent>
                                        </Dialog>
                                    </CardContent>
                                </Card>

                                <!-- Separator -->
                                <div class="flex flex-col items-center justify-center h-full">
                                    <div class="flex flex-col items-center gap-2">
                                        <Separator orientation="vertical" class="h-8" />
                                        <span class="text-sm text-muted-foreground px-4">or</span>
                                        <Separator orientation="vertical" class="h-8" />
                                    </div>
                                </div>

                                <!-- Connect Table Card -->
                                <Card
                                    :class="{ 'border-primary shadow-sm': tableMode === 'connect', 'border-muted-foreground/20': tableMode !== 'connect' }"
                                    class="cursor-pointer transition-all hover:border-primary/70"
                                    @click="tableMode = 'connect' as const">
                                    <CardHeader>
                                        <div class="flex items-center gap-2">
                                            <RadioGroupItem value="connect" id="connect" />
                                            <Label for="connect" class="cursor-pointer font-medium">Connect Existing
                                                Table</Label>
                                        </div>
                                    </CardHeader>
                                    <CardContent class="space-y-4">
                                        <div class="flex items-start gap-4">
                                            <Database class="h-5 w-5 mt-1 text-muted-foreground" />
                                            <div class="space-y-1">
                                                <p class="text-sm font-medium">Use an existing table</p>
                                                <p class="text-sm text-muted-foreground">Connect to an existing
                                                    ClickHouse
                                                    table where your logs are already being ingested. You'll need to
                                                    specify which fields contain the timestamp and severity.</p>
                                            </div>
                                        </div>

                                        <!-- Timestamp and Severity Fields Input -->
                                        <div v-if="tableMode === 'connect'" class="mt-4 border-t pt-4 space-y-4">
                                            <div class="grid gap-2">
                                                <Label for="meta_ts_field" class="required">Timestamp Field Name</Label>
                                                <Input id="meta_ts_field" v-model="metaTSField" placeholder="timestamp"
                                                    required />
                                                <p class="text-sm text-muted-foreground mt-1">
                                                    Specify the field name that contains the timestamp in your table.
                                                    Must
                                                    be of
                                                    type DateTime or DateTime64.
                                                </p>
                                            </div>

                                            <div class="grid gap-2">
                                                <Label for="meta_severity_field">Severity Field
                                                    Name (Optional)</Label>
                                                <Input id="meta_severity_field" v-model="metaSeverityField"
                                                    placeholder="severity_text" />
                                                <p class="text-sm text-muted-foreground mt-1">
                                                    Optionally specify the field name that contains the severity level
                                                    in
                                                    your table. Leave empty if not needed.
                                                </p>
                                            </div>
                                        </div>
                                    </CardContent>
                                </Card>
                            </RadioGroup>
                        </div>

                        <!-- Validation Section -->
                        <div v-if="!createTable" class="space-y-4 border-t pt-4">
                            <div class="flex items-center justify-between">
                                <div class="text-sm font-medium">Validate Connection</div>
                                <Button type="button" variant="outline" @click="handleValidateConnection"
                                    :disabled="isValidating || isValidated" size="sm">
                                    <span v-if="isValidating" class="mr-2">
                                        <svg class="animate-spin h-4 w-4 text-primary"
                                            xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor"
                                                stroke-width="4"></circle>
                                            <path class="opacity-75" fill="currentColor"
                                                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z">
                                            </path>
                                        </svg>
                                    </span>
                                    <span v-else-if="isValidated" class="mr-2">✓</span>
                                    {{ isValidated ? 'Validated' : validateButtonText }}
                                </Button>
                            </div>

                            <!-- Validation Success Result -->
                            <div v-if="validationResult"
                                class="p-3 rounded-md text-sm bg-green-50 text-green-800 border border-green-200">
                                {{ validationResult.message }}
                            </div>
                        </div>
                    </div>

                    <div class="flex justify-end space-x-4 border-t pt-6 mt-6">
                        <Button type="button" variant="outline" @click="router.push({ name: 'Sources' })">
                            Cancel
                        </Button>
                        <Button type="submit" :disabled="isSubmitting || !isValid || (!createTable && !isValidated)"
                            :class="{ 'opacity-50': !isValid || (!createTable && !isValidated) }" size="lg">
                            <span v-if="isSubmitting" class="mr-2">
                                <svg class="animate-spin h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg"
                                    fill="none" viewBox="0 0 24 24">
                                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor"
                                        stroke-width="4"></circle>
                                    <path class="opacity-75" fill="currentColor"
                                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z">
                                    </path>
                                </svg>
                            </span>
                            {{ submitButtonText }}
                        </Button>
                    </div>
                </form>
            </CardContent>
        </Card>
    </div>
</template>

<style scoped>
.required::after {
    content: " *";
    color: hsl(var(--destructive));
}
</style>
