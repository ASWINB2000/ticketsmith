import {templates} from '../../wailsjs/go/models'
import {Input} from '@/components/ui/input'
import {Button} from '@/components/ui/button'
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select'
import {Label} from '@/components/ui/label'
import {InfoTooltip} from '@/components/InfoTooltip'
import {Trash2Icon, PlusIcon} from 'lucide-react'

type Field = templates.Field

interface JsonFieldsEditorProps {
    fields: Field[]
    onChange: (fields: Field[]) => void
}

function emptyField(): Field {
    return {name: '', label: '', type: 'text', description: ''} as Field
}

export function JsonFieldsEditor({fields, onChange}: JsonFieldsEditorProps) {
    const update = (index: number, patch: Partial<Field>) => {
        onChange(fields.map((f, i) => (i === index ? {...f, ...patch} : f)))
    }
    const remove = (index: number) => {
        onChange(fields.filter((_, i) => i !== index))
    }
    const add = () => {
        onChange([...fields, emptyField()])
    }

    return (
        <div className="grid gap-3">
            <Label className="inline-flex w-fit items-center gap-1.5">
                Extraction fields
                <InfoTooltip>
                    Specific pieces of information the AI should pull out of your notes — beyond the subject and
                    description — like "steps to reproduce" or "acceptance criteria". Each one becomes its own
                    editable field in the preview before you submit.
                </InfoTooltip>
            </Label>
            {fields.map((field, index) => (
                <div key={index} className="grid grid-cols-[1fr_1fr_auto_auto] items-end gap-2 rounded-lg border p-2">
                    <div className="grid gap-1">
                        <Label className="text-xs text-muted-foreground">Name</Label>
                        <Input
                            value={field.name}
                            onChange={(e) => update(index, {name: e.target.value})}
                            placeholder="steps_to_reproduce"
                        />
                    </div>
                    <div className="grid gap-1">
                        <Label className="text-xs text-muted-foreground">Label</Label>
                        <Input
                            value={field.label}
                            onChange={(e) => update(index, {label: e.target.value})}
                            placeholder="Steps to reproduce"
                        />
                    </div>
                    <div className="grid gap-1">
                        <Label className="text-xs text-muted-foreground">Type</Label>
                        <Select value={field.type} onValueChange={(v) => update(index, {type: v as string})}>
                            <SelectTrigger>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="text">text</SelectItem>
                                <SelectItem value="textarea">textarea</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                    <Button variant="ghost" size="icon" onClick={() => remove(index)}>
                        <Trash2Icon />
                    </Button>
                </div>
            ))}
            <Button variant="outline" size="sm" onClick={add} className="justify-self-start">
                <PlusIcon /> Add field
            </Button>
        </div>
    )
}
