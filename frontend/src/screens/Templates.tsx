import {useEffect, useState} from 'react'
import {toast} from 'sonner'
import {templates} from '../../wailsjs/go/models'
import {api} from '@/lib/api'
import {Card, CardContent, CardHeader, CardTitle, CardDescription} from '@/components/ui/card'
import {Button} from '@/components/ui/button'
import {Input} from '@/components/ui/input'
import {Textarea} from '@/components/ui/textarea'
import {Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog'
import {FormField} from '@/components/FormField'
import {DataTable, type DataTableColumn} from '@/components/DataTable'
import {ConfirmDialog} from '@/components/ConfirmDialog'
import {JsonFieldsEditor} from '@/components/JsonFieldsEditor'
import {PlusIcon} from 'lucide-react'

type Template = templates.Template
type Field = templates.Field

interface TemplateFormState {
    name: string
    trackerTypeName: string
    aiInstructions: string
    fieldsSchema: Field[]
}

const emptyForm: TemplateFormState = {name: '', trackerTypeName: '', aiInstructions: '', fieldsSchema: []}

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
        {key: 'name', header: 'Name', render: (t) => t.name},
        {key: 'type', header: 'Tracker type', render: (t) => t.trackerTypeName},
        {key: 'fields', header: 'Fields', render: (t) => t.fieldsSchema?.length ?? 0},
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
        <div className="grid gap-6 p-4">
            <Card>
                <CardHeader>
                    <CardTitle>Templates</CardTitle>
                    <CardDescription>Define how freeform notes get turned into structured tickets.</CardDescription>
                </CardHeader>
                <CardContent className="grid gap-4">
                    <DataTable columns={columns} rows={list} rowKey={(t) => t.id} emptyMessage="No templates yet." />
                    <Button onClick={openCreate} className="justify-self-start">
                        <PlusIcon /> Add template
                    </Button>
                </CardContent>
            </Card>

            <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogContent className="max-w-lg">
                    <DialogHeader>
                        <DialogTitle>{editingId ? 'Edit template' : 'Add template'}</DialogTitle>
                    </DialogHeader>
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
                    <DialogFooter>
                        <Button onClick={save} disabled={saving}>{saving ? 'Saving…' : 'Save'}</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    )
}
