
import { ChevronRightIcon } from "@heroicons/react/24/outline";
import {
  ComponentType,
  FunctionComponent,
  ReactNode,
  useEffect,
  useRef,
  useState,
} from "react";

interface CardProps {
  title: string;
  logo: string;
  logo1: string;
  tag?: string;
  onClick?: () => void;
  option?: string
}

const UseCaseCard: FunctionComponent<CardProps> = ({
  title,
  logo,
  logo1,
  tag,
  onClick,
  option
}) => {
  const truncate = (text: string | undefined, number: number) => {
    if (text) {
      return text.length > number ? text.substring(0, number) + "..." : text;
    }
  };

  return (
      <>
          <div
              onClick={() => {
                  onClick?.()
              }}
              className="card cursor-pointer rounded-lg border shadow-2xl dark:border-none dark:bg-white h-full flex flex-col justify-between  w-full gap-4 "
          >
              <div className="flex flex-row justify-between rounded-xl  items-center px-4 py-2">
                  <div className="flex flex-row gap-2">
                      <div className=" bg-gray-300 dark:bg-slate-400 rounded p-2">
                          <img src={logo} className=" h-5 w-5" />
                      </div>
                      <div className="bg-gray-300 dark:bg-slate-400 rounded p-2">
                          <img src={logo1} className=" h-5 w-5" />
                      </div>
                      <div className="bg-gray-300 dark:bg-slate-400 rounded p-2">
                          <img src={logo1} className=" h-5 w-5" />
                      </div>
                  </div>
                  <div>
                      {/* <span className="rounded-3xl text-black dark:text-white bg-gray-300 dark:bg-slate-400 px-3 py-1 text-center">
              {tag}
            </span> */}
                  </div>
              </div>
              <div className=" text-start text-black text-wrap px-4 py-2 ">
                  {title}
              </div>

              <div className="flex flex-row justify-center w-full bg-openg-950 dark:bg-blue-900 rounded-b-lg px-4 py-2 items-center">
                  {/* <span className="dark:text-white">google sheet + some text </span> */}
                  <div className="flex w-full text-white flex-row justify-center items-center gap-2">
                      <span>Run it</span>
                      <ChevronRightIcon className="w-5" color="white" />
                  </div>
              </div>
          </div>
      </>
  )
};

export default UseCaseCard;
