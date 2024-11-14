import { Checkbox, useCheckboxState } from 'pretty-checkbox-react'
import { Button, Flex, Text } from '@tremor/react'
import { useEffect, useState } from 'react'
import { TypesFindingSeverity } from '../../../../../../api/api'
import { compareArrays } from '../../../../../../components/Layout/Header/Filter'

interface ISeverity {
    value: TypesFindingSeverity[] | undefined
    defaultValue: TypesFindingSeverity[]
    condition: string
    onChange: (s: TypesFindingSeverity[]) => void
}

export default function Severity({
    value,
    defaultValue,
    condition,
    onChange,
}: ISeverity) {
    const [con, setCon] = useState(condition)
    const severityCheckbox = useCheckboxState({
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        state: [...value],
    })

    useEffect(() => {
        if (
            !compareArrays(
                value?.sort() || [],
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                severityCheckbox.state.sort()
            ) ||
            con !== condition
        ) {
            if (condition === 'is') {
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                onChange([...severityCheckbox.state])
            }
            if (condition === 'isNot') {
                const arr = defaultValue.filter(
                    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                    // @ts-ignore
                    (x) => !severityCheckbox.state.includes(x)
                )
                onChange(arr)
            }
            setCon(condition)
        }
    }, [severityCheckbox.state, condition])

    const options = [
        {
            name: 'Critical',
            value: TypesFindingSeverity.FindingSeverityCritical,
            color: '#6E120B',
        },
        {
            name: 'High',
            value: TypesFindingSeverity.FindingSeverityHigh,
            color: '#CA2B1D',
        },
        {
            name: 'Medium',
            value: TypesFindingSeverity.FindingSeverityMedium,
            color: '#EE9235',
        },
        {
            name: 'Low',
            value: TypesFindingSeverity.FindingSeverityLow,
            color: '#F4C744',
        },
        {
            name: 'None',
            value: TypesFindingSeverity.FindingSeverityNone,
            color: '#9BA2AE',
        },
    ]

    return (
        <Flex flexDirection="col" alignItems="start" className="gap-1.5">
            {options.map((o) => (
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                <Checkbox
                    shape="curve"
                    className="!items-start w-full"
                    value={o.value}
                    {...severityCheckbox}
                >
                    <Flex className="gap-1.5">
                        <div
                            className="h-4 w-1.5 rounded-sm"
                            style={{
                                backgroundColor: o.color,
                            }}
                        />
                        <Text>{o.name}</Text>
                    </Flex>
                </Checkbox>
            ))}
            {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
            {/* @ts-ignore */}
            {!compareArrays(value?.sort(), defaultValue?.sort()) && (
                <Flex className="pt-3 mt-3 border-t border-t-gray-200">
                    <Button
                        variant="light"
                        onClick={() => {
                            onChange(defaultValue)
                            severityCheckbox.setState(defaultValue)
                        }}
                    >
                        Reset
                    </Button>
                </Flex>
            )}
        </Flex>
    )
}
