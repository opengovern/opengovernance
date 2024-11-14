import { TrashIcon } from '@heroicons/react/24/outline'
import { Radio } from 'pretty-checkbox-react'
import { Button, Card, Flex, Icon, Text } from '@tremor/react'
import DefaultConditionSelector, {
    SelectorOptions,
} from '../ConditionSelector/DefaultConditionSelector'

export interface IRadioSelector {
    title: string
    radioItems: RadioItem[]
    selectedValue: string | undefined
    onItemSelected: (item: RadioItem) => void
    supportedConditions: SelectorOptions[]
    selectedCondition: SelectorOptions
    onConditionChange: (condition: SelectorOptions) => void
    onRemove?: () => void
    onReset?: () => void
}

export interface RadioItem {
    title: string
    icon?: any
    iconAlt?: any
    value: string
}

export default function RadioSelector({
    title,
    radioItems,
    selectedValue,
    supportedConditions,
    selectedCondition,
    onItemSelected,
    onConditionChange,
    onRemove,
    onReset,
}: IRadioSelector) {
    return (
        <Card className="mt-2 py-4 px-6 min-w-[200px] w-fit rounded-xl">
            <Flex>
                <Flex
                    justifyContent="start"
                    alignItems="baseline"
                    className="gap-2"
                >
                    <Text>{title}</Text>

                    <DefaultConditionSelector
                        supportedConditions={supportedConditions}
                        selectedCondition={selectedCondition}
                        onConditionChange={(i) => onConditionChange(i)}
                    />
                </Flex>
                {onRemove && (
                    <TrashIcon
                        className="hover:cursor-pointer w-4 text-gray-400"
                        onClick={() => onRemove()}
                    />
                )}
            </Flex>

            <Flex flexDirection="col" alignItems="start" className="gap-2 my-4">
                {radioItems.map((i) => (
                    <Radio
                        name={title}
                        key={`${title}-${i.value}`}
                        checked={selectedValue === i.value}
                        onClick={() => onItemSelected(i)}
                    >
                        <Flex>
                            {i.icon}
                            {i.iconAlt}

                            <Text className="text-gray-700 whitespace-nowrap">
                                {i.title}
                            </Text>
                        </Flex>
                    </Radio>
                ))}
            </Flex>

            {onReset && (
                <Button variant="light" onClick={() => onReset()}>
                    Reset
                </Button>
            )}
        </Card>
    )
}
