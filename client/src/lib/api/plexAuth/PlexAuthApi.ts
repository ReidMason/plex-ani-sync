import type {ApiResponse} from "@/lib/api/common";
import type {Result} from "@/lib/common";
import {getApiBaseUrl} from "@/lib/env";

interface GetPlexAuthUrlResponse {
  authUrl: string;
}

export const getPlexAuthUrl = async (
  forwardUrl: string
): Promise<Result<GetPlexAuthUrlResponse>> => {
  try {
    const url = new URL(getApiBaseUrl());
    url.searchParams.set("forwardUrl", forwardUrl);
    url.pathname = "/api/plex/authUrl";

    const response = await fetch(url.toString());
    const data = (await response.json()) as ApiResponse<GetPlexAuthUrlResponse>;

    return {
      success: true,
      data: data.data,
    };
  } catch (error) {
    return {
      success: false,
      error: error as Error,
    };
  }
};

export async function getPlexPin(pinId: string) {
  try {
    const url = new URL(getApiBaseUrl());
    url.pathname = `/api/plex/pins/${pinId}`;
    const response = await fetch(url.toString());
    const data = await response.json();

    return {
      success: true,
      data: data,
    };
  } catch (error) {
    return {
      success: false,
      error: error as Error,
    };
  }
}
