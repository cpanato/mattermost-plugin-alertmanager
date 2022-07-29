import {id as pluginId} from './manifest';

import CustomAttributesSettings from './components/admin_settings/CustomAttributeSettings.jsx';

export default class Plugin {
    initialize(registry, store) {

        const CustomAttributesSettingsWrapper = (props) => {

            return (
                <CustomAttributesSettings {...props} store={store} />
            );
        }

        registry.registerAdminConsoleCustomSetting('alertConfigs', CustomAttributesSettingsWrapper);
    }
}

window.registerPlugin(pluginId, new Plugin());
