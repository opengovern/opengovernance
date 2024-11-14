import { Button, Card, Flex } from '@tremor/react'
import { PlusIcon } from '@heroicons/react/20/solid'
import { Popover, Transition } from '@headlessui/react'
import { Fragment } from 'react'
import FilterSingle from './FilterSingle'
import { DateRange } from '../../utilities/urlstate'

interface IFilterGroup {
    filterList: IFilter[]
    addedFilters: string[]
    onFilterAdded: (filterTitle: string) => void
    alignment?: 'left' | 'right'
}

export interface IFilter {
    title: string
    icon: any
    itemsTitles: string[] | undefined
    selector: React.ReactElement
    isValueChanged: boolean
}

export default function FilterGroup({
    filterList,
    addedFilters,
    onFilterAdded,
    alignment = 'left',
}: IFilterGroup) {
    return (
        <Flex justifyContent="start" className="gap-4">
            {addedFilters.length > 0 && (
                <Flex className="w-fit flex-wrap gap-2" justifyContent="start">
                    {filterList.map(
                        (i) =>
                            addedFilters.includes(i.title) && (
                                <FilterSingle
                                    title={i.title}
                                    icon={i.icon}
                                    itemsTitles={i.itemsTitles}
                                    isValueChanged={i.isValueChanged}
                                    alignment={alignment}
                                >
                                    {i.selector}
                                </FilterSingle>
                            )
                    )}
                </Flex>
            )}

            {filterList.filter((f) => !addedFilters.includes(f.title)).length >
                0 && (
                <Popover className="relative border-0">
                    <Popover.Button>
                        <Button
                            variant="light"
                            icon={PlusIcon}
                            className="h-[34px]"
                        >
                            Add Filter
                        </Button>
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
                            className={`absolute z-50 top-full ${alignment}-0`}
                        >
                            <Card className="mt-1 p-4 pr-6 w-fit">
                                <Flex
                                    flexDirection="col"
                                    justifyContent="start"
                                    alignItems="start"
                                    className="gap-1.5 max-w-full"
                                >
                                    {filterList
                                        .filter(
                                            (f) =>
                                                !addedFilters.includes(f.title)
                                        )
                                        .map((f) => (
                                            <Button
                                                icon={f.icon}
                                                color="slate"
                                                variant="light"
                                                className="w-full pl-1 flex justify-start"
                                                onClick={() =>
                                                    onFilterAdded(f.title)
                                                }
                                            >
                                                {f.title}
                                            </Button>
                                        ))}
                                </Flex>
                            </Card>
                        </Popover.Panel>
                    </Transition>
                </Popover>
            )}
        </Flex>
    )
}
