interface Instance {
  uuid: string;
  version: string;
  last_seen: string;
}

interface InstancesResponse {
  instances: Instance[];
  total: number;
  status: number;
}

async function getInstances(apiServer: string): Promise<InstancesResponse> {
  const response = await fetch(`${apiServer}/v1/instances/all`);

  if (!response.ok) {
    console.error("Failed to fetch instances:", response.statusText);
    return { instances: [], total: 0, status: response.status };
  }

  const data = (await response.json()) as InstancesResponse;

  if (data.status !== 200) {
    console.error("API returned error status:", data.status);
    return { instances: [], total: 0, status: data.status };
  }

  return {
    instances: data.instances || [],
    total: data.total || 0,
    status: response.status,
  };
}

async function main(apiServer: string) {
  const instancesResponse = await getInstances(apiServer);
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  var versionCounts: { [key: string]: number } = {};

  console.log(`Using ${timezone} timezone for last seen timestamps.`);

  for (const instance of instancesResponse.instances) {
    versionCounts[instance.version] =
      (versionCounts[instance.version] || 0) + 1;
    const last_seen = new Date(instance.last_seen).toLocaleString(undefined, {
      timeZone: timezone,
    });
    console.log(
      `UUID: ${instance.uuid}, Version: ${instance.version}, Last Seen: ${last_seen}`
    );
  }

  for (const [version, count] of Object.entries(versionCounts)) {
    console.log(`Version: ${version}, Count: ${count}`);
  }

  console.log(`Total instances: ${instancesResponse.total}`);
}

var apiServer = "https://api.tinyauth.app";

const apiServerArg = Bun.argv[2];

if (apiServerArg) {
  apiServer = apiServerArg;
}

console.log(`Using API server: ${apiServer}`);

main(apiServer);
