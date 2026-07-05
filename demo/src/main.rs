use eframe::egui;
use serde::{Deserialize, Serialize};
use std::sync::{Arc, Mutex};

fn main() -> Result<(), eframe::Error> {
    let options = eframe::NativeOptions {
        viewport: egui::ViewportBuilder::default()
            .with_inner_size([400.0, 620.0])
            .with_title("iUpdater 测试工具"),
        ..Default::default()
    };
    
    eframe::run_native(
        "iUpdater 测试工具",
        options,
        Box::new(|cc| {
            setup_custom_fonts(&cc.egui_ctx);
            Box::new(UpdaterApp::default())
        }),
    )
}

fn setup_custom_fonts(ctx: &egui::Context) {
    let mut fonts = egui::FontDefinitions::default();
    
    let font_paths = vec![
        r"C:\Windows\Fonts\msyh.ttc",
        r"C:\Windows\Fonts\simhei.ttf",
        r"C:\Windows\Fonts\simsun.ttc",
    ];
    
    let mut font_loaded = false;
    for font_path in font_paths {
        if let Ok(font_data) = std::fs::read(font_path) {
            fonts.font_data.insert(
                "chinese_font".to_owned(),
                egui::FontData::from_owned(font_data),
            );
            font_loaded = true;
            break;
        }
    }
    
    if font_loaded {
        fonts.families
            .entry(egui::FontFamily::Proportional)
            .or_default()
            .insert(0, "chinese_font".to_owned());
        
        fonts.families
            .entry(egui::FontFamily::Monospace)
            .or_default()
            .insert(0, "chinese_font".to_owned());
    }
    
    ctx.set_fonts(fonts);
}

#[derive(Debug, Clone, Serialize)]
struct CheckUpdateRequest {
    app_id: String,
    version_code: i64,
    channel: String,
    platform: String,
    architecture: String,
    access_key: String,
}

#[derive(Debug, Clone, Deserialize)]
struct CheckUpdateResponse {
    need_update: bool,
    is_force_update: bool,
    latest_version: VersionInfo,
    components: Vec<ComponentInfo>,
}

#[derive(Debug, Clone, Deserialize, Default)]
#[allow(dead_code)]
struct VersionInfo {
    app_id: String,
    version_code: i64,
    version_name: String,
    channel: String,
    platform: String,
    architecture: String,
    package_type: String,
    file_size: i64,
    file_hash: String,
    download_url: String,
    release_notes: String,
}

#[derive(Debug, Clone, Deserialize)]
struct ComponentInfo {
    component_key: String,
    name: String,
    version_code: i64,
    version_name: String,
    file_size: i64,
    file_hash: String,
    download_url: String,
}

#[derive(Debug, Clone, PartialEq)]
enum UpdateTab {
    FullPackage,
    Components,
}

struct UpdaterApp {
    server_url: String,
    app_id: String,
    current_version_code: String,
    channel: String,
    access_key: String,
    check_result: Arc<Mutex<Option<Result<CheckUpdateResponse, String>>>>,
    is_checking: Arc<Mutex<bool>>,
    active_tab: UpdateTab,
    component_versions: Arc<Mutex<std::collections::HashMap<String, String>>>,
    runtime: tokio::runtime::Runtime,
}

impl Default for UpdaterApp {
    fn default() -> Self {
        Self {
            server_url: "http://localhost:8080".to_string(),
            app_id: "com.example.test".to_string(),
            current_version_code: "202607050001".to_string(),
            channel: "stable".to_string(),
            access_key: "".to_string(),
            check_result: Arc::new(Mutex::new(None)),
            is_checking: Arc::new(Mutex::new(false)),
            active_tab: UpdateTab::FullPackage,
            component_versions: Arc::new(Mutex::new(std::collections::HashMap::new())),
            runtime: tokio::runtime::Runtime::new().unwrap(),
        }
    }
}

impl UpdaterApp {
    fn check_update(&mut self) {
        let version_code = self.current_version_code.parse::<i64>().unwrap_or(0);
        let request = CheckUpdateRequest {
            app_id: self.app_id.clone(),
            version_code,
            channel: self.channel.clone(),
            platform: "windows".to_string(),
            architecture: "x86_64".to_string(),
            access_key: self.access_key.clone(),
        };
        
        let url = format!("{}/api/check-update", self.server_url);
        let check_result = self.check_result.clone();
        let is_checking = self.is_checking.clone();
        
        *is_checking.lock().unwrap() = true;
        *check_result.lock().unwrap() = None;
        
        self.runtime.spawn(async move {
            let result = async {
                let client = reqwest::Client::new();
                let response = client.post(&url).json(&request).send().await
                    .map_err(|e| format!("请求失败: {}", e))?;
                
                if !response.status().is_success() {
                    let status = response.status();
                    let body = response.text().await.unwrap_or_default();
                    return Err(format!("服务器错误: {} - {}", status, body));
                }
                
                response.json().await.map_err(|e| format!("解析失败: {}", e))
            }.await;
            
            *check_result.lock().unwrap() = Some(result);
            *is_checking.lock().unwrap() = false;
        });
    }
    
    fn format_size(bytes: i64) -> String {
        if bytes <= 0 {
            return "-".to_string();
        }
        if bytes < 1024 {
            format!("{} B", bytes)
        } else if bytes < 1024 * 1024 {
            format!("{:.2} KB", bytes as f64 / 1024.0)
        } else if bytes < 1024 * 1024 * 1024 {
            format!("{:.2} MB", bytes as f64 / (1024.0 * 1024.0))
        } else {
            format!("{:.2} GB", bytes as f64 / (1024.0 * 1024.0 * 1024.0))
        }
    }
    
    fn make_full_url(&self, url: &str) -> String {
        if url.starts_with('/') {
            format!("{}{}", self.server_url.trim_end_matches('/'), url)
        } else {
            url.to_string()
        }
    }
    
    fn truncate_hash(hash: &str) -> String {
        if hash.is_empty() {
            "-".to_string()
        } else if hash.len() > 16 {
            hash[..16].to_string() + "..."
        } else {
            hash.to_string()
        }
    }
    
    fn component_needs_update(&self, comp: &ComponentInfo) -> Option<bool> {
        self.component_versions.lock().unwrap()
            .get(&comp.component_key)
            .and_then(|v| v.parse::<i64>().ok())
            .map(|current| comp.version_code > current)
    }
}

impl eframe::App for UpdaterApp {
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        egui::CentralPanel::default().show(ctx, |ui| {
            ui.heading("iUpdater Demo");
            ui.add_space(10.0);
            
            // 配置区
            egui::Grid::new("config").num_columns(2).spacing([10.0, 8.0]).show(ui, |ui| {
                ui.label("服务器:");
                ui.text_edit_singleline(&mut self.server_url);
                ui.end_row();
                
                ui.label("App ID:");
                ui.text_edit_singleline(&mut self.app_id);
                ui.end_row();
                
                ui.label("当前版本号:");
                ui.text_edit_singleline(&mut self.current_version_code);
                ui.end_row();
                
                ui.label("Access Key:");
                ui.text_edit_singleline(&mut self.access_key);
                ui.end_row();
            });
            
            ui.add_space(10.0);
            ui.horizontal(|ui| {
                ui.label("渠道:");
                ui.radio_value(&mut self.channel, "stable".to_string(), "正式版");
                ui.radio_value(&mut self.channel, "beta".to_string(), "测试版");
            });
            
            ui.add_space(10.0);
            let is_checking = *self.is_checking.lock().unwrap();
            if ui.add_enabled(!is_checking, egui::Button::new("检查更新")).clicked() {
                self.check_update();
            }
            
            if is_checking {
                ui.horizontal(|ui| {
                    ui.spinner();
                    ui.label("正在检查更新...");
                });
            }
            
            ui.add_space(10.0);
            ui.separator();
            ui.add_space(10.0);
            
            // 结果显示区
            if let Some(result) = self.check_result.lock().unwrap().as_ref() {
                match result {
                    Ok(response) => {
                        // 顶部状态摘要
                        if response.need_update {
                            let text = if response.is_force_update { "发现强制更新" } else { "发现新版本" };
                            ui.colored_label(egui::Color32::from_rgb(0, 150, 0), text);
                        } else {
                            ui.colored_label(egui::Color32::from_rgb(100, 100, 100), "已是最新版本");
                        }
                        
                        ui.add_space(10.0);
                        
                        // 选项卡
                        let has_components = !response.components.is_empty();
                        
                        ui.horizontal(|ui| {
                            let full_tab = ui.selectable_label(self.active_tab == UpdateTab::FullPackage, "完整包");
                            if full_tab.clicked() {
                                self.active_tab = UpdateTab::FullPackage;
                            }
                            
                            let comp_count = response.components.len();
                            let comp_label = if comp_count > 0 {
                                format!("组件 ({})", comp_count)
                            } else {
                                "组件".to_string()
                            };
                            let comp_tab = ui.add_enabled(has_components, egui::SelectableLabel::new(
                                self.active_tab == UpdateTab::Components, comp_label));
                            if comp_tab.clicked() {
                                self.active_tab = UpdateTab::Components;
                            }
                        });
                        
                        ui.separator();
                        ui.add_space(5.0);
                        
                        // 根据选项卡显示不同内容
                        match self.active_tab {
                            UpdateTab::FullPackage => {
                                self.render_full_package(ui, response);
                            }
                            UpdateTab::Components => {
                                self.render_components(ui, response);
                            }
                        }
                    }
                    Err(err) => {
                        ui.colored_label(egui::Color32::from_rgb(200, 0, 0), format!("错误: {}", err));
                    }
                }
            }
        });
        
        // 持续请求重绘以更新 UI 状态
        if *self.is_checking.lock().unwrap() || self.check_result.lock().unwrap().is_some() {
            ctx.request_repaint();
        }
    }
}

impl UpdaterApp {
    fn render_full_package(&self, ui: &mut egui::Ui, response: &CheckUpdateResponse) {
        let version = &response.latest_version;
        
        egui::Frame::none()
            .fill(egui::Color32::from_gray(240))
            .inner_margin(10.0)
            .rounding(5.0)
            .show(ui, |ui| {
                ui.label(egui::RichText::new("版本信息").strong());
                ui.add_space(5.0);
                
                egui::Grid::new("version_info").num_columns(2).spacing([10.0, 5.0]).show(ui, |ui| {
                    ui.label("版本号:");
                    ui.label(format!("{} ({})", version.version_name, version.version_code));
                    ui.end_row();
                    
                    ui.label("渠道:");
                    let channel_text = if version.channel == "beta" { "测试版" } else { "正式版" };
                    ui.label(channel_text);
                    ui.end_row();
                    
                    ui.label("平台:");
                    ui.label(format!("{} / {}", version.platform, version.architecture));
                    ui.end_row();
                    
                    ui.label("类型:");
                    ui.label(&version.package_type);
                    ui.end_row();
                    
                    ui.label("大小:");
                    ui.label(Self::format_size(version.file_size));
                    ui.end_row();
                    
                    ui.label("SHA256:");
                    ui.label(Self::truncate_hash(&version.file_hash));
                    ui.end_row();
                });
                
                if !version.release_notes.is_empty() {
                    ui.add_space(5.0);
                    ui.label(egui::RichText::new("更新说明:").strong());
                    ui.label(&version.release_notes);
                }
                
                ui.add_space(5.0);
                let full_url = self.make_full_url(&version.download_url);
                ui.horizontal(|ui| {
                    ui.label(egui::RichText::new("下载链接:").strong());
                    ui.label(egui::RichText::new(&full_url).small().weak());
                });
                if ui.button("复制下载链接").clicked() {
                    ui.output_mut(|o| o.copied_text = full_url);
                }
            });
    }
    
    fn render_components(&self, ui: &mut egui::Ui, response: &CheckUpdateResponse) {
        if response.components.is_empty() {
            ui.label(egui::RichText::new("暂无组件").weak());
            return;
        }
        
        // 组件版本管理区
        ui.label(egui::RichText::new("本地组件版本（用于检测是否需要更新）").small().weak());
        ui.add_space(5.0);
        
        egui::ScrollArea::vertical().max_height(180.0).show(ui, |ui| {
            for comp in &response.components {
                egui::Frame::none()
                    .fill(egui::Color32::from_gray(245))
                    .inner_margin(8.0)
                    .rounding(4.0)
                    .show(ui, |ui| {
                        ui.horizontal(|ui| {
                            ui.label(egui::RichText::new(&comp.name).strong());
                            ui.label(egui::RichText::new(format!("({})", comp.component_key)).small().weak());
                            
                            // 状态标记
                            let needs_update = self.component_needs_update(comp);
                            match needs_update {
                                Some(true) => {
                                    ui.colored_label(egui::Color32::from_rgb(200, 120, 0), "需更新");
                                }
                                Some(false) => {
                                    ui.colored_label(egui::Color32::from_rgb(0, 130, 0), "最新");
                                }
                                None => {
                                    ui.label(egui::RichText::new("未设置").small().weak());
                                }
                            }
                        });
                        
                        ui.horizontal(|ui| {
                            ui.label("服务器版本:");
                            ui.label(format!("{} ({})", comp.version_name, comp.version_code));
                        });
                        
                        ui.horizontal(|ui| {
                            ui.label("本地版本:");
                            let current = self.component_versions.lock().unwrap()
                                .get(&comp.component_key)
                                .cloned()
                                .unwrap_or_default();
                            let mut current_edit = current.clone();
                            ui.add(egui::TextEdit::singleline(&mut current_edit).desired_width(120.0));
                            if current_edit != current {
                                self.component_versions.lock().unwrap()
                                    .insert(comp.component_key.clone(), current_edit);
                            }
                        });
                        
                        ui.horizontal(|ui| {
                            ui.label("大小:");
                            ui.label(Self::format_size(comp.file_size));
                            ui.label("  SHA256:");
                            ui.label(Self::truncate_hash(&comp.file_hash));
                        });
                        
                        let comp_full_url = self.make_full_url(&comp.download_url);
                        ui.horizontal(|ui| {
                            ui.label(egui::RichText::new("下载:").small());
                            ui.label(egui::RichText::new(&comp_full_url).small().weak());
                        });
                        
                        if ui.small_button("复制链接").clicked() {
                            ui.output_mut(|o| o.copied_text = comp_full_url);
                        }
                    });
                ui.add_space(4.0);
            }
        });
    }
}
