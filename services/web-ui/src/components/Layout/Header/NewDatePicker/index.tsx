import dayjs from 'dayjs'
import { Popover, Transition } from '@headlessui/react'
import { Card, Flex, Text } from '@tremor/react'
import { CalendarIcon, ChevronDownIcon } from '@heroicons/react/24/outline'
import { Fragment, useState } from 'react'
import { DateRange, useUrlDateRangeState } from '../../../../utilities/urlstate'
import { renderDateText } from '../DatePicker'
import ConditionDropdown from '../../../ConditionDropdown'
import Datepicker from './Datepicker'

interface INewDatePicker {
    defaultTime: DateRange
}
export default function NewDatePicker({ defaultTime }: INewDatePicker) {
    const [condition, setCondition] = useState('isBetween')
    const { value: activeTimeRange, setValue: setActiveTimeRange } =
        useUrlDateRangeState(defaultTime)

    return (
        <Popover className="relative border-0">
            <Popover.Button
                id="timepicker"
                className="border border-openg-500 text-openg-500 bg-openg-50 py-1.5 px-2 rounded-md"
            >
                <Flex className="w-fit gap-2">
                    <CalendarIcon className="w-4 text-inherit" />
                    <Text className="text-inherit w-fit whitespace-nowrap">
                        {renderDateText(
                            activeTimeRange.start,
                            activeTimeRange.end
                        )}
                    </Text>
                    <ChevronDownIcon className="w-3 text-inherit" />
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
                <Popover.Panel
                    static
                    className="absolute z-50 top-full right-0"
                >
                    <Card className="mt-2 p-4 min-w-[256px] w-fit">
                        <Flex className="w-fit gap-1.5">
                            <Text className="font-semibold">Date</Text>
                            <ConditionDropdown
                                onChange={(c) => setCondition(c)}
                                conditions={['isBetween', 'isRelative']}
                                isDate
                            />
                        </Flex>
                        <Datepicker
                            condition={condition}
                            defaultTime={defaultTime}
                        />
                    </Card>
                </Popover.Panel>
            </Transition>
        </Popover>
    )
}
