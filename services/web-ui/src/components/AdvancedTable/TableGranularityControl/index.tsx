import { Select, SelectItem, Switch, Text } from '@tremor/react'
import { Dispatch, SetStateAction } from 'react'
import { capitalizeFirstLetter } from '../../../utilities/labelMaker'

interface IProps {
    selectedGranularity: 'daily' | 'monthly'
    onGranularityChange: Dispatch<SetStateAction<'monthly' | 'daily'>>
}

export default function TableGranularityControl({
    selectedGranularity,
    onGranularityChange,
}: IProps) {
    return (
        <>
            <label htmlFor="switch" className="text-sm">
                Spend Granularity{' '}
            </label>
            <Select
                enableClear={false}
                value={selectedGranularity}
                placeholder={capitalizeFirstLetter(selectedGranularity)}
                onValueChange={(v) => {
                    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                    // @ts-ignore
                    onGranularityChange(v)
                }}
                className="w-10"
            >
                <SelectItem value="daily">
                    <Text>Daily</Text>
                </SelectItem>
                <SelectItem value="monthly">
                    <Text>Monthly</Text>
                </SelectItem>
            </Select>
        </>
    )
}
