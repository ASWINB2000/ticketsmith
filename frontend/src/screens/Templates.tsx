import {useEffect, useState} from 'react'
import {toast} from 'sonner'
import {templates} from '../../wailsjs/go/models'
import {api} from '@/lib/api'
import {Card, CardContent, CardHeader, CardTitle, CardDescription} from '@/components/ui/card'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Textarea} from '@/components/ui/textarea'
import {Dialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogDescription} from '@/components/ui/dialog'
import {FormField} from '@/components/FormField'
import {DataTable, type DataTableColumn} from '@/components/DataTable'
import {ConfirmDialog} from '@/components/ConfirmDialog'
import {JsonFieldsEditor} from '@/components/JsonFieldsEditor'
import {PageHeader} from '@/components/Layout/PageHeader'
import {Badge} from '@/components/ui/badge'
import {PlusIcon, EyeIcon, LayoutTemplateIcon} from 'lucide-react'

type Template = templates.Template
type Field = templates.Field

interface TemplateFormState {
    name: string
    trackerTypeName: string
    aiInstructions: string
    fieldsSchema: Field[]
}

const emptyForm: TemplateFormState = {name: '', trackerTypeName: '', aiInstructions: '', fieldsSchema: []}

function samplePlaceholder(field: Field): string {
    if (field.description) return field.description
    return `Example ${(field.label || field.name || 'value').toLowerCase()}…`
}

function TemplatePreview({form}: { form: TemplateFormState }) {
    return (
        <div className="grid gap-4 rounded-lg border bg-muted/40 p-4">
            <p className="text-xs text-muted-foreground">
                A rough sketch of what you'll see on the Generate screen after the AI turns your notes into a{' '}
                <span className="font-medium text-foreground">{form.trackerTypeName || 'ticket'}</span>.
            </p>
            <FormField label="Subject">
                <Input disabled value={`Sample subject for a ${form.trackerTypeName || 'ticket'}`} />
            </FormField>
            <FormField label="Description">
                <Textarea disabled rows={3} value="The AI-generated summary of your notes will appear here, ready to edit before you submit." />
            </FormField>
            {form.fieldsSchema.length === 0 ? (
                <p className="rounded-md border border-dashed p-3 text-center text-xs text-muted-foreground">
                    No extraction fields yet — add some below to see them here.
                </p>
            ) : (
                form.fieldsSchema.map((f, i) => (
                    <FormField key={i} label={f.label || f.name || 'Untitled field'}>
                        {f.type === 'textarea' ? (
                            <Textarea disabled rows={2} value={samplePlaceholder(f)} />
                        ) : (
                            <Input disabled value={samplePlaceholder(f)} />
                        )}
                    </FormField>
                ))
            )}
        </div>
    )
}

export function Templates() {
    const [list, setList] = useState<Template[]>([])
    const [dialogOpen, setDialogOpen] = useState(false)
    const [editingId, setEditingId] = useState<string | null>(null)
    const [form, setForm] = useState<TemplateFormState>(emptyForm)
    const [saving, setSaving] = useState(false)

    const refresh = () => {
        api.templates.list().then(setList).catch((err) => toast.error(`Failed to load templates: ${err}`))
    }

    useEffect(refresh, [])

    const openCreate = () => {
        setEditingId(null)
        setForm(emptyForm)
        setDialogOpen(true)
    }

    const openEdit = (t: Template) => {
        setEditingId(t.id)
        setForm({
            name: t.name,
            trackerTypeName: t.trackerTypeName,
            aiInstructions: t.aiInstructions,
            fieldsSchema: t.fieldsSchema ?? [],
        })
        setDialogOpen(true)
    }

    const save = async () => {
        setSaving(true)
        try {
            const payload = new templates.Template({
                id: editingId ?? '',
                name: form.name,
                trackerTypeName: form.trackerTypeName,
                aiInstructions: form.aiInstructions,
                fieldsSchema: form.fieldsSchema,
            })
            if (editingId) {
                await api.templates.update(payload)
                toast.success('Template updated')
            } else {
                await api.templates.create(payload)
                toast.success('Template created')
            }
            setDialogOpen(false)
            refresh()
        } catch (err) {
            toast.error(`${err}`)
        } finally {
            setSaving(false)
        }
    }

    const remove = async (id: string) => {
        try {
            await api.templates.remove(id)
            toast.success('Template deleted')
            refresh()
        } catch (err) {
            toast.error(`${err}`)
        }
    }

    const columns: DataTableColumn<Template>[] = [
        {key: 'name', header: 'Name', render: (t) => <span className="font-medium">{t.name}</span>},
        {key: 'type', header: 'Tracker type', render: (t) => <Badge variant="outline">{t.trackerTypeName}</Badge>},
        {
            key: 'fields',
            header: 'Fields',
            render: (t) => <span className="text-muted-foreground">{t.fieldsSchema?.length ?? 0}</span>,
        },
        {
            key: 'actions',
            header: '',
            className: 'text-right',
            render: (t) => (
                <div className="flex justify-end gap-2">
                    <Button variant="outline" size="sm" onClick={() => openEdit(t)}>Edit</Button>
                    <ConfirmDialog
                        trigger={<Button variant="destructive" size="sm">Delete</Button>}
                        title={`Delete "${t.name}"?`}
                        description="This cannot be undone."
                        confirmLabel="Delete"
                        destructive
                        onConfirm={() => remove(t.id)}
                    />
                </div>
            ),
        },
    ]

    return (
        <div className="flex flex-col">
            <PageHeader
                icon={LayoutTemplateIcon}
                title="Templates"
                description="Define how freeform notes get turned into structured tickets."
                actions={
                    <Button onClick={openCreate}>
                        <PlusIcon /> Add template
                    </Button>
                }
            />
            <div className="p-8">
                <Card>
                    <CardHeader>
                        <CardTitle>Your templates</CardTitle>
                        <CardDescription>Each template maps to a tracker type and the fields the AI should extract.</CardDescription>
                    </CardHeader>
                    <CardContent>
                        <DataTable columns={columns} rows={list} rowKey={(t) => t.id} emptyMessage="No templates yet." />
                    </CardContent>
                </Card>
            </div>

            <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogContent className="flex max-h-[85vh] w-full flex-col sm:max-w-3xl lg:max-w-5xl">
                    <DialogHeader>
                        <DialogTitle>{editingId ? 'Edit template' : 'Add template'}</DialogTitle>
                        <DialogDescription>Maps freeform notes to a tracker type and the fields the AI should extract.</DialogDescription>
                    </DialogHeader>
                    <div className="-mx-4 grid min-h-0 flex-1 gap-6 overflow-y-auto px-4 py-1 lg:grid-cols-[minmax(0,1fr)_320px]">
                        <div className="grid gap-4">
                            <FormField label="Name" htmlFor="tmpl-name">
                                <Input id="tmpl-name" value={form.name} onChange={(e) => setForm((f) => ({...f, name: e.target.value}))} />
                            </FormField>
                            <FormField
                                label="Tracker type name"
                                htmlFor="tmpl-type"
                                description="Must match a type name in your tracker (e.g. Bug, Task, User story)."
                            >
                                <Input
                                    id="tmpl-type"
                                    value={form.trackerTypeName}
                                    onChange={(e) => setForm((f) => ({...f, trackerTypeName: e.target.value}))}
                                />
                            </FormField>
                            <FormField label="AI instructions" htmlFor="tmpl-instructions">
                                <Textarea
                                    id="tmpl-instructions"
                                    rows={4}
                                    value={form.aiInstructions}
                                    onChange={(e) => setForm((f) => ({...f, aiInstructions: e.target.value}))}
                                    placeholder="Extract a clear subject and description. Be concise and specific."
                                />
                            </FormField>
                            <JsonFieldsEditor
                                fields={form.fieldsSchema}
                                onChange={(fields) => setForm((f) => ({...f, fieldsSchema: fields}))}
                            />
                        </div>
                        <div className="grid content-start gap-2 lg:sticky lg:top-0">
                            <span className="inline-flex w-fit items-center gap-1.5 text-xs font-medium text-muted-foreground uppercase tracking-wide">
                                <EyeIcon className="size-3.5" /> Live preview
                            </span>
                            <TemplatePreview form={form} />
                        </div>
                    </div>
                    <DialogFooter>
                        <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
                        <Button onClick={save} disabled={saving}>{saving ? 'Saving…' : 'Save'}</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    )
}
