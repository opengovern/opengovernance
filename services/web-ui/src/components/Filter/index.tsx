import { Button, Card, Flex, Icon, Text, TextInput } from '@tremor/react'
import { MagnifyingGlassIcon } from '@heroicons/react/24/outline'
import { Checkbox, useCheckboxState } from 'pretty-checkbox-react'
import { useEffect, useState } from 'react'
import Spinner from '../Spinner'
import { Popover, Transition } from '@headlessui/react'
import { Fragment, ComponentType } from 'react'
import ConditionDropdown from '../ConditionDropdown'
import {
    CalendarIcon,
    CheckCircleIcon,
    ChevronDownIcon,
    CloudIcon,
    PlusIcon,
    TrashIcon,
} from '@heroicons/react/24/outline'
// import {IFilter} from './types'
interface Option {
    label: string | number | undefined
    value: string | number | undefined
    showValue?: boolean
    icon? : ComponentType<any>
    color?: string

}

export interface IFilter {
    options: Option[]
    type: 'multi' | 'single'
    label: string
    onChange: Function
    selectedItems: string[]
    icon: ComponentType<any>
    onDelete?: Function
    hasCondition?: boolean
    condition?: string
    defaultValue?: Option
}

export default function KFilter({
    options,
    type,
    label,
    onChange,
    selectedItems,
    icon,
    condition,
    hasCondition,
}: IFilter) {
    const [search, setSearch] = useState('')
    const checkbox = useCheckboxState({ state: [...selectedItems] })
    const [con, setCon] = useState<string>(hasCondition == true && condition ? condition : 'is')
    useEffect(() => {
        // @ts-ignore
        if (hasCondition == true) {
            if (condition === 'is') {
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                onChange([...checkbox.state])
            }
            if (condition === 'isNot') {
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                const arr = options
                    .filter(
                        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                        // @ts-ignore
                        (x) => !checkbox.state.includes(x.key)
                    )
                    .map((x) => x.value)
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                onChange(arr)
            }
            setCon(condition ? condition : 'is')
        } else {
            // @ts-ignore
            onChange([...checkbox.state])
        }
    }, [checkbox.state, con])
    return (
        <Popover className="relative border-0">
            <Popover.Button
                id={'salam'}
                className={`border   py-1.5 px-2 rounded-md  
                ${
                    selectedItems.length == 0
                        ? 'border-gray-200 bg-white'
                        : 'border-openg-500 text-openg-500 bg-openg-50'
                }
                   `}
            >
                <Flex className="w-fit">
                    <Icon icon={icon} className="w-3 p-0 mr-3 text-inherit" />
                    <Text className="text-inherit whitespace-nowrap">
                        {label}
                        {selectedItems.length > 0 && (
                            <>
                                {' : '}
                                {selectedItems.length == 1
                                    ? options.filter((item, value) => {
                                          return item.value == selectedItems[0]
                                      })[0]?.label
                                    : `( ${selectedItems.length} )`}
                            </>
                        )}
                    </Text>
                    <ChevronDownIcon className="ml-1 w-3 text-inherit" />
                </Flex>
            </Popover.Button>
            <Transition
                as={Fragment}
                enter="transition ease-out duration-200"
                enterFrom="opacity-0 translate-y-1"
                enterTo="opacity-100 translate-y-0"
                leave="transition ease-in duration-150"
                leaveFrom="opacity-100 translate-y-0"
                leaveTo="opacity-0 translate-y-1"
            >
                <Popover.Panel static className="absolute z-50 top-full left-0">
                    <Card className="mt-2 p-4 min-w-[256px] w-fit">
                        <Flex className="mb-3">
                            <Flex className="w-fit gap-1.5">
                                <Text className="font-semibold">{label}</Text>
                                {hasCondition == true && (
                                    <>
                                        <ConditionDropdown
                                            onChange={(c) => setCon(c)}
                                            conditions={[con]}
                                            isDate={false}
                                        />
                                    </>
                                )}
                            </Flex>
                            {/* {f?.onDelete && (
                                    <div className="group relative">
                                        <TrashIcon
                                            onClick={() => {
                                                f?.onDelete()
                                                setSelectedFilters(
                                                    (prevState) => {
                                                        return prevState.filter(
                                                            (s) => s !== f?.id
                                                        )
                                                    }
                                                )
                                            }}
                                            className="w-4 cursor-pointer hover:text-openg-500"
                                        />
                                        <Card className="absolute w-fit z-40 -top-2 left-full ml-2 scale-0 transition-all p-2 group-hover:scale-100">
                                            <Text className="whitespace-nowrap">
                                                Remove filter
                                            </Text>
                                        </Card>
                                    </div>
                                )} */}
                        </Flex>
                        <Flex
                            flexDirection="col"
                            justifyContent="start"
                            alignItems="start"
                        >
                            <TextInput
                                icon={MagnifyingGlassIcon}
                                placeholder="Search..."
                                value={search}
                                onChange={(e) => setSearch(e.target.value)}
                                className="mb-4"
                            />
                            <Flex
                                flexDirection="col"
                                justifyContent="start"
                                alignItems="start"
                                className="gap-1.5 max-h-[200px] overflow-y-scroll no-scroll max-w-full"
                            >
                                {options ? (
                                    options
                                        ?.filter(
                                            (d) =>
                                                d?.label
                                                    ?.toString()
                                                    ?.toLowerCase()
                                                    .includes(
                                                        search.toLowerCase()
                                                    ) ||
                                                d?.value
                                                    ?.toString()
                                                    ?.toLowerCase()
                                                    .includes(
                                                        search.toLowerCase()
                                                    )
                                        )
                                        .map(
                                            (d, i) =>
                                                i < 100 && (
                                                    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                                    // @ts-ignore
                                                    <Checkbox
                                                        shape="curve"
                                                        className="!items-start"
                                                        value={d.value}
                                                        {...checkbox}
                                                    >
                                                        <Flex
                                                            flexDirection="col"
                                                            alignItems="start"
                                                        >
                                                            <Text className="text-gray-800 truncate">
                                                                {d.label}
                                                            </Text>
                                                            {d.showValue ==
                                                                true && (
                                                                <>
                                                                    <Text className="text-xs truncate max-w-[200px]">
                                                                        {
                                                                            d.value
                                                                        }
                                                                    </Text>
                                                                </>
                                                            )}
                                                        </Flex>
                                                    </Checkbox>
                                                )
                                        )
                                ) : (
                                    <Spinner />
                                )}
                            </Flex>
                            {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
                            {/* @ts-ignore */}
                            {selectedItems && selectedItems.length !== 0 && (
                                <Flex className="pt-3 mt-3 border-t border-t-gray-200">
                                    <Button
                                        variant="light"
                                        onClick={() => {
                                            onChange([])
                                            checkbox.setState([])
                                        }}
                                    >
                                        Reset
                                    </Button>
                                </Flex>
                            )}
                        </Flex>
                    </Card>
                </Popover.Panel>
            </Transition>
        </Popover>
    )
}
