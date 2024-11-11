import { useState } from 'react'
import { Button, Card, List, ListItem } from '@tremor/react'
import { ChevronDownIcon } from '@heroicons/react/20/solid'
import { camelCaseToLabel } from '../../../../utilities/labelMaker'

interface IConditionDropdown {
    supportedConditions: SelectorOptions[] | undefined
    selectedCondition: SelectorOptions | undefined
    onConditionChange: (c: SelectorOptions) => void
}

export type SelectorOptions =
    | 'is'
    | 'isNot'
    | 'contains'
    | 'doesNotContain'
    | 'isEmpty'
    | 'isNotEmpty'

export default function DefaultConditionSelector({
    supportedConditions = ['is'],
    selectedCondition = 'is',
    onConditionChange,
}: IConditionDropdown) {
    const [open, setOpen] = useState(false)

    return (
        <div className="relative z-10">
            <Button
                variant="light"
                icon={ChevronDownIcon}
                iconPosition="right"
                size="xs"
                onClick={() => setOpen(!open)}
            >
                {camelCaseToLabel(selectedCondition).toLowerCase()}
            </Button>

            {open && (
                <Card className="mt-2 px-2 py-1 absolute w-36">
                    <List>
                        {supportedConditions.map((o) => (
                            <ListItem key={o}>
                                <Button
                                    variant="light"
                                    color={
                                        o === selectedCondition
                                            ? 'blue'
                                            : 'slate'
                                    }
                                    onClick={() => {
                                        onConditionChange(o)
                                        setOpen(false)
                                    }}
                                    className="w-full flex justify-start"
                                >
                                    {camelCaseToLabel(o).toLowerCase()}
                                </Button>
                            </ListItem>
                        ))}
                    </List>
                </Card>
            )}
        </div>
    )
}
